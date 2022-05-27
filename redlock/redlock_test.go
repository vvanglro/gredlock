package redlock

import (
	"context"
	"fmt"
	red "github.com/go-redis/redis/v8"
	"testing"
)

func CommonClient() []*RedClient {
	var clients []*RedClient
	ctx := context.Background()
	client := red.NewClient(&red.Options{
		Network: "tcp",
		Addr:    "localhost:56379",
	})
	lockSha := client.ScriptLoad(ctx, lockCommand).Val()
	delSha := client.ScriptLoad(ctx, delCommand).Val()
	ttlSha := client.ScriptLoad(ctx, ttlCommand).Val()
	clients = append(clients, &RedClient{cli: client, lockSha: lockSha, delSha: delSha, ttlSha: ttlSha})
	return clients
}

func TestNewRedisLock(t *testing.T) {
	type args struct {
		ctx     context.Context
		options []*red.Options
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test redis connect",
			args: args{ctx: context.Background(), options: []*red.Options{
				{
					Network: "tcp",
					Addr:    "localhost:56379",
				},
				{
					Network: "tcp",
					Addr:    "localhost:56378",
				},
			}},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRedisLock(tt.args.ctx, tt.args.options...)
			for _, client := range got.client {
				err := client.cli.Ping(ctx).Err()
				if err != nil {
					t.Fatalf("redis connect error")
				}
			}
		})
	}
}

func TestRedLock_SetLock(t *testing.T) {
	type fields struct {
		client []*RedClient
	}
	type args struct {
		ctx   context.Context
		key   string
		value string
		ttl   int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test set lock",
			fields: fields{
				client: CommonClient(),
			},
			args: args{
				ctx:   context.Background(),
				key:   "my-key",
				value: "123",
				ttl:   60,
			},
			want:    "float64",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RedLock{
				client: tt.fields.client,
			}
			got, err := rl.SetLock(tt.args.ctx, tt.args.key, tt.args.value, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if fmt.Sprintf("%T", got) != tt.want {
				t.Errorf("SetLock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedLock_GetLockTtl(t *testing.T) {
	type fields struct {
		client []*RedClient
	}
	type args struct {
		ctx   context.Context
		key   string
		value string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "test get lock ttl",
			fields: fields{
				client: CommonClient(),
			},
			args: args{
				ctx:   context.Background(),
				key:   "my-key",
				value: "123",
			},
			want:    60,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RedLock{
				client: tt.fields.client,
			}
			got, err := rl.GetLockTtl(tt.args.ctx, tt.args.key, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLockTtl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetLockTtl() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedLock_IsLocked(t *testing.T) {
	type fields struct {
		client []*RedClient
	}
	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test key IsLocked",
			fields: fields{client: CommonClient()},
			args: args{
				ctx: context.Background(),
				key: "my-key",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RedLock{
				client: tt.fields.client,
			}
			got := rl.IsLocked(tt.args.ctx, tt.args.key)
			if got != tt.want {
				t.Errorf("IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedLock_UnSetLock(t *testing.T) {
	type fields struct {
		client []*RedClient
	}
	type args struct {
		ctx   context.Context
		key   string
		value string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test unset lock",
			fields: fields{
				client: CommonClient(),
			},
			args: args{
				ctx:   context.Background(),
				key:   "my-key",
				value: "123",
			},
			want:    "float64",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RedLock{
				client: tt.fields.client,
			}
			got, err := rl.UnSetLock(tt.args.ctx, tt.args.key, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnSetLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if fmt.Sprintf("%T", got) != tt.want {
				t.Errorf("UnSetLock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedLock_allEqual(t *testing.T) {
	type fields struct {
		client []*RedClient
	}
	type args struct {
		iter []int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test allEqual",
			fields: fields{
				client: CommonClient(),
			},
			args: args{
				iter: []int64{1, 1, 1},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RedLock{
				client: tt.fields.client,
			}
			if got := rl.allEqual(tt.args.iter); got != tt.want {
				t.Errorf("allEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedLock_sum(t *testing.T) {
	type fields struct {
		client []*RedClient
	}
	type args struct {
		iter []bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: " test sum",
			fields: fields{
				client: CommonClient(),
			},
			args: args{
				iter: []bool{true, false, true},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := &RedLock{
				client: tt.fields.client,
			}
			if got := rl.sum(tt.args.iter); got != tt.want {
				t.Errorf("sum() = %v, want %v", got, tt.want)
			}
		})
	}
}
