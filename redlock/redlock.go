package redlock

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	red "github.com/go-redis/redis/v8"
)

const (
	tolerance       = 500 // milliseconds
	millisPerSecond = 1000
)

var (
	lockCommand = `
    local identifier = redis.call('get', KEYS[1])
    if not identifier or identifier == ARGV[1] then
        return redis.call("set", KEYS[1], ARGV[1], 'PX', ARGV[2])
    else
        return redis.error_reply('ERROR')
    end`
	delCommand = ` local identifier = redis.call('get', KEYS[1])
    if not identifier then
        return redis.status_reply('OK')
    elseif identifier == ARGV[1] then
        return redis.call("del", KEYS[1])
    else
        return redis.error_reply('ERROR')
    end`
	ttlCommand = `local identifier = redis.call('get', KEYS[1])
	if not identifier then
	return redis.error_reply('ERROR')
	elseif identifier == ARGV[1] then
	return redis.call("TTL", KEYS[1])
	else
	return redis.error_reply('ERROR')
	end`
)

type RedClient struct {
	cli     *red.Client
	lockSha string
	delSha  string
	ttlSha  string
}

type RedLock struct {
	client []*RedClient
}

// NewRedisLock returns a RedisLock.
func NewRedisLock(ctx context.Context, options ...*red.Options) *RedLock {
	var clients []*RedClient
	for _, opt := range options {
		client := red.NewClient(opt)
		lockSha := client.ScriptLoad(ctx, lockCommand).Val()
		delSha := client.ScriptLoad(ctx, delCommand).Val()
		ttlSha := client.ScriptLoad(ctx, ttlCommand).Val()
		clients = append(clients, &RedClient{cli: client, lockSha: lockSha, delSha: delSha, ttlSha: ttlSha})
	}

	return &RedLock{
		client: clients,
	}
}

func (rl *RedLock) sum(iter []bool) int {
	num := 0
	for i := 0; i < len(iter); i++ {
		if iter[i] {
			num++
		}
	}
	return num
}

func (rl *RedLock) allEqual(iter []int64) bool {
	for i := 1; i < len(iter); i++ {
		if iter[i] != iter[0] {
			return false
		}
	}
	return true
}

// SetLock acquires the lock.
func (rl *RedLock) SetLock(ctx context.Context, key string, value string, ttl int) (float64, error) {
	startTime := time.Now()
	var wg sync.WaitGroup
	var successfulSets []bool
	for _, client := range rl.client {
		wg.Add(1)
		client := client
		go func() {
			defer wg.Done()
			resp, err := client.cli.EvalSha(ctx, client.lockSha, []string{key},
				value, strconv.Itoa(int(ttl)*millisPerSecond+tolerance)).Result()
			if err != nil {
				successfulSets = append(successfulSets, false)
				return
			} else if resp == nil {
				successfulSets = append(successfulSets, false)
				return
			}

			reply, ok := resp.(string)
			if ok && reply == "OK" {
				successfulSets = append(successfulSets, true)
				return
			}
			successfulSets = append(successfulSets, true)
			return
		}()
	}
	wg.Wait()
	successfulNum := rl.sum(successfulSets)
	elapsed := fmt.Sprintf("%.2f", time.Now().Sub(startTime).Seconds())
	float, _ := strconv.ParseFloat(elapsed, 64)
	locked := successfulNum >= int(len(rl.client)/2)+1
	if locked != true {
		return 0, errors.New(fmt.Sprintf("Can not acquire the lock %s", key))
	}
	return float, nil
}

// UnSetLock releases the lock.
func (rl *RedLock) UnSetLock(ctx context.Context, key string, value string) (float64, error) {
	startTime := time.Now()
	var wg sync.WaitGroup
	var successfulSets []bool
	for _, client := range rl.client {
		wg.Add(1)
		client := client
		go func() {
			defer wg.Done()
			resp, err := client.cli.EvalSha(ctx, client.delSha, []string{key}, value).Result()
			if err != nil {
				successfulSets = append(successfulSets, false)
				return
			}
			reply, ok := resp.(string)
			if ok && reply == "OK" {
				successfulSets = append(successfulSets, true)
				return
			}
			successfulSets = append(successfulSets, true)
			return
		}()
	}
	wg.Wait()
	successfulNum := rl.sum(successfulSets)
	elapsed := fmt.Sprintf("%.2f", time.Now().Sub(startTime).Seconds())
	float, _ := strconv.ParseFloat(elapsed, 64)
	locked := successfulNum >= int(len(rl.client)/2)+1
	if locked != true {
		return 0, errors.New(fmt.Sprintf("Can not release the lock %s", key))
	}
	return float, nil
}

// GetLockTtl time the lock.
func (rl *RedLock) GetLockTtl(ctx context.Context, key string, value string) (int64, error) {
	startTime := time.Now()
	var wg sync.WaitGroup
	var successfulSets []bool
	var TtlSets []int64
	for _, client := range rl.client {
		wg.Add(1)
		client := client
		go func() {
			defer wg.Done()
			resp, err := client.cli.EvalSha(ctx, client.ttlSha, []string{key}, value).Result()
			if err != nil {
				successfulSets = append(successfulSets, false)
				return
			}

			reply, _ := resp.(int64)
			TtlSets = append(TtlSets, reply)
			successfulSets = append(successfulSets, true)
			return
		}()
	}
	wg.Wait()
	successfulNum := rl.sum(successfulSets)
	locked := successfulNum >= int(len(rl.client)/2)+1
	success := rl.allEqual(TtlSets) && locked
	elapsed := fmt.Sprintf("%.2f", time.Now().Sub(startTime).Seconds())
	float, _ := strconv.ParseFloat(elapsed, 64)
	if success != true {
		return 0, errors.New(fmt.Sprintf("Could not fetch the TTL for lock %s in %.2f seconds", key, float))
	}
	return TtlSets[0], nil
}

func (rl *RedLock) IsLocked(ctx context.Context, key string) bool {
	var wg sync.WaitGroup
	var successfulSets []bool
	for _, client := range rl.client {
		wg.Add(1)
		client := client
		go func() {
			defer wg.Done()
			_, err := client.cli.Get(ctx, key).Result()
			if err != nil {
				successfulSets = append(successfulSets, false)
				return
			}
			successfulSets = append(successfulSets, true)
			return
		}()
	}
	wg.Wait()
	successfulNum := rl.sum(successfulSets)
	return successfulNum >= int(len(rl.client)/2)+1
}
