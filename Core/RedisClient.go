package Core

//go get github.com/go-redis/redis/
//go get github.com/go-redis/redis/v8
//go clean -i github.com/go-redis/redis/
//go get github.com/ulule/limiter/v3
//go get github.com/ulule/limiter/v3/drivers/store/redis

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	redisStore "github.com/ulule/limiter/v3/drivers/store/redis"
	shared "httpRequestName/Shared"
	"log"
	"time"
)

type RedisClient struct {
	c *redis.Client
}

var client = &RedisClient{}
var ctx = context.Background()

func GetRedisClient() *RedisClient {

	//Test SqlDatabase
	//database, _ := Decrypt(shared.Config.SQLURL, shared.Config.SECRETKEY)
	//encdatabase, _ := Encrypt("sqlserver://sa:AspNet55@192.168.50.173:1433?database=Customer&encrypt=disable&connection+timeout=30", shared.Config.SECRETKEY)
	//fmt.Println(database)
	//fmt.Println(encdatabase)

	redisPassword, _ := Decrypt(shared.Config.REDISPASSWORD, shared.Config.SECRETKEY)
	redisUrl, _ := Decrypt(shared.Config.REDISURL, shared.Config.SECRETKEY)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: redisPassword, // no password set
		DB:       0,             // use default DB
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		panic("Unable to connect to redis " + err.Error())
	}
	client.c = rdb
	return client
}

func (client *RedisClient) SetKey(key string, value interface{}, expiration time.Duration) error {
	cacheData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = client.c.Set(ctx, key, cacheData, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (client *RedisClient) GetKey(key string, src interface{}) error {
	val, err := client.c.Get(ctx, key).Result()
	if err == redis.Nil || err != nil {
		return err
	}

	err = json.Unmarshal([]byte(val), &src)
	if err != nil {
		return err
	}
	return nil
}

func (client *RedisClient) GetLifeTimeMinutes(key string) (int64, error) {
	ttl, err := client.c.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if ttl <= 0 {
		return 0, nil // Key yoksa ya da süresizse 0 döndür
	}
	return int64(ttl.Minutes()), nil
}

func GenerateRedisKey(userName string, isRefreshToken bool) string {
	if isRefreshToken {
		return userName + ":RefreshToken"
	}
	return userName + ":Token"
}

func GetGlobalLimiter() *limiter.Limiter {
	var client = GetRedisClient()
	rate, err := limiter.NewRateFromFormatted("100-M") // Dakikada 100 istek
	//rate, err := limiter.NewRateFromFormatted("10-S") //Saniyede 10 istek
	if err != nil {
		log.Fatal(err)
	}

	store, err := redisStore.NewStoreWithOptions(client.c, limiter.StoreOptions{
		Prefix:   "rate_limiter",
		MaxRetry: 3,
	})
	if err != nil {
		log.Fatal(err)
	}

	globalLimiter := limiter.New(store, rate)
	return globalLimiter
}
