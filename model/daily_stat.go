package model

import (
	"fmt"
	"strconv"
	"time"

	"just-vpn/pkg/mapping"
	"just-vpn/pkg/redis"

	goredis "github.com/go-redis/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	dailyStatRedisTTL  = 7 * 24 * time.Hour
	dailyStatBatchSize = 1000
)

// DailyStat 每日统计表
type DailyStat struct {
	BaseModel
	StatDate         time.Time `json:"stat_date" gorm:"column:stat_date;type:date;uniqueIndex:uk_daily_stat_date_product;comment:统计日期"`
	ProductCode      string    `json:"product_code" gorm:"column:product_code;type:varchar(32);uniqueIndex:uk_daily_stat_date_product;comment:产品编码"`
	DailyActiveUsers int       `json:"daily_active_users" gorm:"column:daily_active_users;type:int;not null;default:0;comment:日活用户数"`
	DailyNewUsers    int       `json:"daily_new_users" gorm:"column:daily_new_users;type:int;not null;default:0;comment:新增用户数"`
	D1RetentionUsers int       `json:"d1_retention_users" gorm:"column:d1_retention_users;type:int;not null;default:0;comment:次日留存用户数"`
	D1RetentionRate  float64   `json:"d1_retention_rate" gorm:"column:d1_retention_rate;type:decimal(6,2);not null;default:0.00;comment:次日留存率"`
	D2RetentionUsers int       `json:"d2_retention_users" gorm:"column:d2_retention_users;type:int;not null;default:0;comment:2日留存用户数"`
	D2RetentionRate  float64   `json:"d2_retention_rate" gorm:"column:d2_retention_rate;type:decimal(6,2);not null;default:0.00;comment:2日留存率"`
	D3RetentionUsers int       `json:"d3_retention_users" gorm:"column:d3_retention_users;type:int;not null;default:0;comment:3日留存用户数"`
	D3RetentionRate  float64   `json:"d3_retention_rate" gorm:"column:d3_retention_rate;type:decimal(6,2);not null;default:0.00;comment:3日留存率"`
}

func (DailyStat) TableName() string {
	return "daily_stat"
}

func RecordDailyActive(userId int) error {
	return addDailyStatUser(dailyActiveKey(chinaStatDate(time.Now())), userId)
}

func RecordDailyNewUser(userId int) error {
	now := chinaStatDate(time.Now())
	if err := addDailyStatUser(dailyNewKey(now), userId); err != nil {
		return err
	}
	return addDailyStatUser(dailyActiveKey(now), userId)
}

func DailyNewUserIDsFromCache(statDate time.Time) ([]int, bool, error) {
	key := dailyNewKey(chinaStatDate(statDate))
	exists, err := redisKeyExists(key)
	if err != nil || !exists {
		return nil, exists, err
	}
	return redisSetIntMembers(key)
}

func SyncDailyStatForDate(statDate time.Time) error {
	statDate = chinaStatDate(statDate)
	activeKey := dailyActiveKey(statDate)
	newKey := dailyNewKey(statDate)
	hasActiveKey, err := redisKeyExists(activeKey)
	if err != nil {
		return err
	}
	hasNewKey, err := redisKeyExists(newKey)
	if err != nil {
		return err
	}
	if !hasActiveKey && !hasNewKey {
		return nil
	}
	activeUsers, err := redisSetCount(activeKey)
	if err != nil {
		return err
	}
	newUsers, err := redisSetCount(newKey)
	if err != nil {
		return err
	}
	return upsertDailyStat(DailyStat{
		StatDate:         statDate,
		ProductCode:      mapping.ProductCode(),
		DailyActiveUsers: activeUsers,
		DailyNewUsers:    newUsers,
	})
}

func SyncDailyRetentionByActiveDate(activeDate time.Time) error {
	activeDate = chinaStatDate(activeDate)
	for day := 1; day <= 3; day++ {
		baseDate := activeDate.AddDate(0, 0, -day)
		retentionUsers, err := redisSetIntersectCount(dailyNewKey(baseDate), dailyActiveKey(activeDate))
		if err != nil {
			return err
		}
		newUsers, err := redisSetCount(dailyNewKey(baseDate))
		if err != nil {
			return err
		}
		if err := updateDailyRetention(baseDate, day, retentionUsers, retentionRate(retentionUsers, newUsers)); err != nil {
			return err
		}
	}
	return nil
}

func backfillUserIds(key string, scope func(tx *gorm.DB) *gorm.DB) (int, error) {
	if redis.Redis == nil {
		return 0, nil
	}
	var lastId int
	total := 0
	for {
		var ids []int
		if err := scope(DB).
			Where("id > ?", lastId).
			Order("id ASC").
			Limit(dailyStatBatchSize).
			Pluck("id", &ids).Error; err != nil {
			return total, err
		}
		if len(ids) == 0 {
			break
		}
		if err := addDailyStatUsers(key, ids); err != nil {
			return total, err
		}
		total += len(ids)
		lastId = ids[len(ids)-1]
		if len(ids) < dailyStatBatchSize {
			break
		}
	}
	return total, nil
}

func addDailyStatUser(key string, userId int) error {
	if redis.Redis == nil {
		return nil
	}
	pipe := redis.Redis.TxPipeline()
	pipe.SAdd(key, userId)
	pipe.Expire(key, dailyStatRedisTTL)
	_, err := pipe.Exec()
	return err
}

func addDailyStatUsers(key string, userIds []int) error {
	if redis.Redis == nil || len(userIds) == 0 {
		return nil
	}
	values := make([]interface{}, 0, len(userIds))
	for _, userId := range userIds {
		values = append(values, userId)
	}
	pipe := redis.Redis.TxPipeline()
	pipe.SAdd(key, values...)
	pipe.Expire(key, dailyStatRedisTTL)
	_, err := pipe.Exec()
	return err
}

func redisSetCount(key string) (int, error) {
	if redis.Redis == nil {
		return 0, nil
	}
	count, err := redis.Redis.SCard(key).Result()
	if err != nil && err != goredis.Nil {
		return 0, err
	}
	return int(count), nil
}

func redisSetIntersectCount(leftKey string, rightKey string) (int, error) {
	if redis.Redis == nil {
		return 0, nil
	}
	tempKey := fmt.Sprintf("stat:%s:retention:tmp:%d", mapping.ProductCode(), time.Now().UnixNano())
	count, err := redis.Redis.SInterStore(tempKey, leftKey, rightKey).Result()
	if err != nil {
		return 0, err
	}
	_ = redis.Redis.Expire(tempKey, time.Minute).Err()
	return int(count), nil
}

func redisSetIntMembers(key string) ([]int, bool, error) {
	if redis.Redis == nil {
		return nil, false, nil
	}
	userIds := make([]int, 0)
	var cursor uint64
	for {
		values, nextCursor, err := redis.Redis.SScan(key, cursor, "", dailyStatBatchSize).Result()
		if err != nil {
			return nil, true, err
		}
		for _, value := range values {
			userId, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			userIds = append(userIds, userId)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return userIds, true, nil
}

func upsertDailyStat(stat DailyStat) error {
	now := time.Now().In(time.Local)
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "stat_date"},
			{Name: "product_code"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"daily_active_users": stat.DailyActiveUsers,
			"daily_new_users":    stat.DailyNewUsers,
			"update_time":        now,
		}),
	}).Create(&stat).Error
}

func redisKeyExists(key string) (bool, error) {
	if redis.Redis == nil {
		return false, nil
	}
	exists, err := redis.Redis.Exists(key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func updateDailyRetention(statDate time.Time, day int, users int, rate float64) error {
	updates := map[string]interface{}{}
	switch day {
	case 1:
		updates["d1_retention_users"] = users
		updates["d1_retention_rate"] = rate
	case 2:
		updates["d2_retention_users"] = users
		updates["d2_retention_rate"] = rate
	case 3:
		updates["d3_retention_users"] = users
		updates["d3_retention_rate"] = rate
	}
	if len(updates) == 0 {
		return nil
	}
	return DB.Model(&DailyStat{}).
		Where("stat_date = ? AND product_code = ?", statDate, mapping.ProductCode()).
		Updates(updates).Error
}

func retentionRate(retentionUsers int, newUsers int) float64 {
	if newUsers <= 0 {
		return 0
	}
	return float64(retentionUsers) * 100 / float64(newUsers)
}

func chinaStatDate(t time.Time) time.Time {
	t = t.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func dailyActiveKey(date time.Time) string {
	return fmt.Sprintf("stat:%s:dau:%s", mapping.ProductCode(), date.Format("20060102"))
}

func dailyNewKey(date time.Time) string {
	return fmt.Sprintf("stat:%s:new:%s", mapping.ProductCode(), date.Format("20060102"))
}
