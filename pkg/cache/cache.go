package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

func Connect(addr string) {
	Client = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
}

func Ping(ctx context.Context) error {
	return Client.Ping(ctx).Err()
}

func Close() error {
	return Client.Close()
}

func Get(ctx context.Context, key string, dest interface{}) error {
	val, err := Client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, b, ttl).Err()
}

func Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return Client.Del(ctx, keys...).Err()
}

func DeletePattern(ctx context.Context, pattern string) error {
	iter := Client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return Client.Del(ctx, keys...).Err()
	}
	return nil
}

// Keys patterns for HMS
const (
	KeyPatientList    = "patients:%s:list:%s:%d:%d"    // tenant, status, page, pageSize
	KeyPatientGet    = "patients:%s:id:%s"             // tenant, patientID
	KeyClinicList    = "clinics:%s:list"                // tenant
	KeyStats        = "stats:%s:%s"                    // tenant, clinic
	KeyFieldConfig  = "patients:%s:fieldConfig:%s"    // tenant, clinic
	KeyUser         = "users:%s:%s"                    // tenant, userID
	KeyCatalogue    = "catalogue:%s:%s"                // tenant, kind
)

func PatientListKey(tenant, status string, page, pageSize int) string {
	return fmt.Sprintf(KeyPatientList, tenant, status, page, pageSize)
}

func PatientGetKey(tenant, patientID string) string {
	return fmt.Sprintf(KeyPatientGet, tenant, patientID)
}

func ClinicListKey(tenant string) string {
	return fmt.Sprintf(KeyClinicList, tenant)
}

func StatsKey(tenant, clinic string) string {
	return fmt.Sprintf(KeyStats, tenant, clinic)
}

func FieldConfigKey(tenant, clinic string) string {
	return fmt.Sprintf(KeyFieldConfig, tenant, clinic)
}

func UserKey(tenant, userID string) string {
	return fmt.Sprintf(KeyUser, tenant, userID)
}

func CatalogueKey(tenant, kind string) string {
	return fmt.Sprintf(KeyCatalogue, tenant, kind)
}

// Invalidate helpers
func InvalidatePatientList(ctx context.Context, tenant string) error {
	return DeletePattern(ctx, "patients:"+tenant+":list:*")
}

func InvalidatePatient(ctx context.Context, tenant, patientID string) error {
	return Delete(ctx, PatientGetKey(tenant, patientID), PatientListKey(tenant, "", 1, 10))
}

func InvalidateClinic(ctx context.Context, tenant string) error {
	return DeletePattern(ctx, "clinics:"+tenant+":list:*")
}