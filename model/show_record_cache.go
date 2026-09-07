package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"just-vpn/pkg/redis"

	goredis "github.com/go-redis/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	showRecordCacheTTL       = 7 * 24 * time.Hour
	showRecordFlushInterval  = time.Minute
	showRecordFlushBatchSize = int64(1000)

	showRecordFieldTimes    = "show_times"
	showRecordFieldLastTime = "last_show_time"

	popupShowRecordPrefix  = "popup:show"
	advertShowRecordPrefix = "advert:show"
	popupDailyShowPrefix   = "popup:daily_show"
)

var removeCleanShowRecordScript = goredis.NewScript(`
if redis.call("HGET", KEYS[1], "show_times") == ARGV[1] and redis.call("HGET", KEYS[1], "last_show_time") == ARGV[2] then
	return redis.call("SREM", KEYS[2], KEYS[1])
end
return 0
`)

var acquirePopupDailyShowScript = goredis.NewScript(`
local current = tonumber(redis.call("GET", KEYS[1]) or "0")
local limit = tonumber(ARGV[1])
if current >= limit then
	return 0
end
redis.call("INCR", KEYS[1])
redis.call("PEXPIRE", KEYS[1], ARGV[2])
return 1
`)

var releasePopupDailyShowScript = goredis.NewScript(`
local current = tonumber(redis.call("GET", KEYS[1]) or "0")
if current <= 0 then
	return 0
end
if current == 1 then
	redis.call("DEL", KEYS[1])
	return 0
end
return redis.call("DECR", KEYS[1])
`)

type showRecordCacheType struct {
	prefix   string
	dirtyKey string
}

var (
	popupShowCache  = showRecordCacheType{prefix: popupShowRecordPrefix, dirtyKey: popupShowRecordPrefix + ":dirty"}
	advertShowCache = showRecordCacheType{prefix: advertShowRecordPrefix, dirtyKey: advertShowRecordPrefix + ":dirty"}
)

func ShowRecordFlushInterval() time.Duration {
	return showRecordFlushInterval
}

func UserPopupCanShow(userId int, popup Popup) (UserPopup, bool, error) {
	record, err := getUserPopupShowRecord(userId, popup.Id)
	if err != nil {
		return UserPopup{}, false, err
	}
	if popup.MaxShowTimes > 0 && record.ShowTimes >= popup.MaxShowTimes {
		return record, false, nil
	}
	return record, true, nil
}

func AcquirePopupDailyShow(popup Popup, now time.Time) (bool, error) {
	if popup.MaxDailyShowTimes <= 0 {
		return true, nil
	}
	if redis.Redis == nil {
		return false, fmt.Errorf("redis is not initialized")
	}
	result, err := acquirePopupDailyShowScript.Run(
		redis.Redis,
		[]string{popupDailyShowKey(popup.Id, now)},
		popup.MaxDailyShowTimes,
		popupDailyShowTTL(now).Milliseconds(),
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func ReleasePopupDailyShow(popup Popup, now time.Time) error {
	if popup.MaxDailyShowTimes <= 0 || redis.Redis == nil {
		return nil
	}
	return releasePopupDailyShowScript.Run(
		redis.Redis,
		[]string{popupDailyShowKey(popup.Id, now)},
	).Err()
}

func popupDailyShowKey(popupId int, now time.Time) string {
	return fmt.Sprintf("%s:%s:%d", popupDailyShowPrefix, now.In(time.Local).Format("20060102"), popupId)
}

func popupDailyShowTTL(now time.Time) time.Duration {
	localNow := now.In(time.Local)
	nextDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, 0, 0, 0, 0, time.Local)
	return nextDay.Sub(localNow) + time.Hour
}

func AddUserPopupShow(userId int, popupId int, now time.Time) (UserPopup, error) {
	record, err := incrementUserPopupShowRecord(userId, popupId, now)
	if err == nil {
		return record, nil
	}
	return addUserPopupShowToDB(userId, popupId, now)
}

func UserAdvertCanShow(userId int, advert Advert) (UserAdvert, bool, error) {
	record, err := getUserAdvertShowRecord(userId, advert.Id)
	if err != nil {
		return UserAdvert{}, false, err
	}
	if advert.MaxShowTimes > 0 && record.ShowTimes >= advert.MaxShowTimes {
		return record, false, nil
	}
	return record, true, nil
}

func AddUserAdvertShow(userId int, advertId int, now time.Time) (UserAdvert, error) {
	record, err := incrementUserAdvertShowRecord(userId, advertId, now)
	if err == nil {
		return record, nil
	}
	return addUserAdvertShowToDB(userId, advertId, now)
}

func FlushShowRecordCaches() error {
	if err := flushPopupShowRecords(); err != nil {
		return err
	}
	return flushAdvertShowRecords()
}

func getUserPopupShowRecord(userId int, popupId int) (UserPopup, error) {
	cacheRecord, ok, err := getShowRecordFromCache(popupShowCache, userId, popupId)
	if err == nil && ok {
		return UserPopup{
			UserId:       userId,
			PopupId:      popupId,
			ShowTimes:    cacheRecord.showTimes,
			LastShowTime: cacheRecord.lastShowTime(),
		}, nil
	}

	record, dbErr := getUserPopupRecordFromDB(userId, popupId)
	if dbErr != nil {
		if !isRecordNotFound(dbErr) {
			return UserPopup{}, dbErr
		}
		record = UserPopup{UserId: userId, PopupId: popupId}
	}
	_ = setShowRecordCache(popupShowCache, userId, popupId, record.ShowTimes, record.LastShowTime)
	return record, nil
}

func incrementUserPopupShowRecord(userId int, popupId int, now time.Time) (UserPopup, error) {
	if err := ensurePopupShowRecordCache(userId, popupId); err != nil {
		return UserPopup{}, err
	}
	record, err := incrementShowRecordCache(popupShowCache, userId, popupId, now)
	if err != nil {
		return UserPopup{}, err
	}
	return UserPopup{
		UserId:       userId,
		PopupId:      popupId,
		ShowTimes:    record.showTimes,
		LastShowTime: record.lastShowTime(),
	}, nil
}

func getUserAdvertShowRecord(userId int, advertId int) (UserAdvert, error) {
	cacheRecord, ok, err := getShowRecordFromCache(advertShowCache, userId, advertId)
	if err == nil && ok {
		return UserAdvert{
			UserId:       userId,
			AdvertId:     advertId,
			ShowTimes:    cacheRecord.showTimes,
			LastShowTime: cacheRecord.lastShowTime(),
		}, nil
	}

	record, dbErr := getUserAdvertRecordFromDB(userId, advertId)
	if dbErr != nil {
		if !isRecordNotFound(dbErr) {
			return UserAdvert{}, dbErr
		}
		record = UserAdvert{UserId: userId, AdvertId: advertId}
	}
	_ = setShowRecordCache(advertShowCache, userId, advertId, record.ShowTimes, record.LastShowTime)
	return record, nil
}

func incrementUserAdvertShowRecord(userId int, advertId int, now time.Time) (UserAdvert, error) {
	if err := ensureAdvertShowRecordCache(userId, advertId); err != nil {
		return UserAdvert{}, err
	}
	record, err := incrementShowRecordCache(advertShowCache, userId, advertId, now)
	if err != nil {
		return UserAdvert{}, err
	}
	return UserAdvert{
		UserId:       userId,
		AdvertId:     advertId,
		ShowTimes:    record.showTimes,
		LastShowTime: record.lastShowTime(),
	}, nil
}

func ensurePopupShowRecordCache(userId int, popupId int) error {
	if _, ok, err := getShowRecordFromCache(popupShowCache, userId, popupId); err != nil || ok {
		return err
	}
	record, err := getUserPopupRecordFromDB(userId, popupId)
	if err != nil {
		if !isRecordNotFound(err) {
			return err
		}
		record = UserPopup{UserId: userId, PopupId: popupId}
	}
	return setShowRecordCache(popupShowCache, userId, popupId, record.ShowTimes, record.LastShowTime)
}

func ensureAdvertShowRecordCache(userId int, advertId int) error {
	if _, ok, err := getShowRecordFromCache(advertShowCache, userId, advertId); err != nil || ok {
		return err
	}
	record, err := getUserAdvertRecordFromDB(userId, advertId)
	if err != nil {
		if !isRecordNotFound(err) {
			return err
		}
		record = UserAdvert{UserId: userId, AdvertId: advertId}
	}
	return setShowRecordCache(advertShowCache, userId, advertId, record.ShowTimes, record.LastShowTime)
}

type showRecordCacheValue struct {
	showTimes    int
	lastShowUnix int64
}

func (record showRecordCacheValue) lastShowTime() *time.Time {
	if record.lastShowUnix <= 0 {
		return nil
	}
	lastShowTime := time.Unix(record.lastShowUnix, 0).In(time.Local)
	return &lastShowTime
}

func getShowRecordFromCache(cacheType showRecordCacheType, userId int, itemId int) (showRecordCacheValue, bool, error) {
	if redis.Redis == nil {
		return showRecordCacheValue{}, false, fmt.Errorf("redis is not initialized")
	}
	values, err := redis.Redis.HMGet(showRecordKey(cacheType, userId, itemId), showRecordFieldTimes, showRecordFieldLastTime).Result()
	if err != nil {
		return showRecordCacheValue{}, false, err
	}
	if len(values) < 2 || values[0] == nil {
		return showRecordCacheValue{}, false, nil
	}
	showTimes, err := parseRedisInt(values[0])
	if err != nil {
		return showRecordCacheValue{}, false, err
	}
	lastShowUnix, err := parseRedisInt64(values[1])
	if err != nil {
		return showRecordCacheValue{}, false, err
	}
	return showRecordCacheValue{showTimes: showTimes, lastShowUnix: lastShowUnix}, true, nil
}

func setShowRecordCache(cacheType showRecordCacheType, userId int, itemId int, showTimes int, lastShowTime *time.Time) error {
	if redis.Redis == nil {
		return fmt.Errorf("redis is not initialized")
	}
	lastShowUnix := int64(0)
	if lastShowTime != nil {
		lastShowUnix = lastShowTime.Unix()
	}
	key := showRecordKey(cacheType, userId, itemId)
	pipe := redis.Redis.TxPipeline()
	pipe.HMSet(key, map[string]interface{}{
		showRecordFieldTimes:    showTimes,
		showRecordFieldLastTime: lastShowUnix,
	})
	pipe.Expire(key, showRecordCacheTTL)
	_, err := pipe.Exec()
	return err
}

func incrementShowRecordCache(cacheType showRecordCacheType, userId int, itemId int, now time.Time) (showRecordCacheValue, error) {
	if redis.Redis == nil {
		return showRecordCacheValue{}, fmt.Errorf("redis is not initialized")
	}
	key := showRecordKey(cacheType, userId, itemId)
	lastShowUnix := now.Unix()
	pipe := redis.Redis.TxPipeline()
	incr := pipe.HIncrBy(key, showRecordFieldTimes, 1)
	pipe.HSet(key, showRecordFieldLastTime, lastShowUnix)
	pipe.Expire(key, showRecordCacheTTL)
	pipe.SAdd(cacheType.dirtyKey, key)
	if _, err := pipe.Exec(); err != nil {
		return showRecordCacheValue{}, err
	}
	return showRecordCacheValue{showTimes: int(incr.Val()), lastShowUnix: lastShowUnix}, nil
}

func showRecordKey(cacheType showRecordCacheType, userId int, itemId int) string {
	return fmt.Sprintf("%s:%d:%d", cacheType.prefix, userId, itemId)
}

func parseRedisInt(value interface{}) (int, error) {
	valueInt64, err := parseRedisInt64(value)
	return int(valueInt64), err
}

func parseRedisInt64(value interface{}) (int64, error) {
	if value == nil {
		return 0, nil
	}
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		if typed == "" {
			return 0, nil
		}
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		if len(typed) == 0 {
			return 0, nil
		}
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return strconv.ParseInt(fmt.Sprint(typed), 10, 64)
	}
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func flushPopupShowRecords() error {
	return flushShowRecords(popupShowCache, func(userId int, itemId int, record showRecordCacheValue) error {
		lastShowTime := record.lastShowTime()
		return DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "popup_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"show_times":     record.showTimes,
				"last_show_time": lastShowTime,
			}),
		}).Create(&UserPopup{
			UserId:       userId,
			PopupId:      itemId,
			ShowTimes:    record.showTimes,
			LastShowTime: lastShowTime,
		}).Error
	})
}

func flushAdvertShowRecords() error {
	return flushShowRecords(advertShowCache, func(userId int, itemId int, record showRecordCacheValue) error {
		lastShowTime := record.lastShowTime()
		return DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "advert_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"show_times":     record.showTimes,
				"last_show_time": lastShowTime,
			}),
		}).Create(&UserAdvert{
			UserId:       userId,
			AdvertId:     itemId,
			ShowTimes:    record.showTimes,
			LastShowTime: lastShowTime,
		}).Error
	})
}

func flushShowRecords(cacheType showRecordCacheType, save func(userId int, itemId int, record showRecordCacheValue) error) error {
	if redis.Redis == nil {
		return nil
	}
	keys, err := redis.Redis.SRandMemberN(cacheType.dirtyKey, showRecordFlushBatchSize).Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		userId, itemId, ok := parseShowRecordKey(cacheType, key)
		if !ok {
			_ = redis.Redis.SRem(cacheType.dirtyKey, key).Err()
			continue
		}
		record, ok, err := getShowRecordByKey(key)
		if err != nil {
			return err
		}
		if !ok {
			_ = redis.Redis.SRem(cacheType.dirtyKey, key).Err()
			continue
		}
		if err := save(userId, itemId, record); err != nil {
			return err
		}
		if err := removeCleanShowRecordDirty(cacheType, key, record); err != nil {
			return err
		}
	}
	return nil
}

func getShowRecordByKey(key string) (showRecordCacheValue, bool, error) {
	values, err := redis.Redis.HMGet(key, showRecordFieldTimes, showRecordFieldLastTime).Result()
	if err != nil {
		return showRecordCacheValue{}, false, err
	}
	if len(values) < 2 || values[0] == nil {
		return showRecordCacheValue{}, false, nil
	}
	showTimes, err := parseRedisInt(values[0])
	if err != nil {
		return showRecordCacheValue{}, false, err
	}
	lastShowUnix, err := parseRedisInt64(values[1])
	if err != nil {
		return showRecordCacheValue{}, false, err
	}
	return showRecordCacheValue{showTimes: showTimes, lastShowUnix: lastShowUnix}, true, nil
}

func removeCleanShowRecordDirty(cacheType showRecordCacheType, key string, record showRecordCacheValue) error {
	return removeCleanShowRecordScript.Run(redis.Redis, []string{key, cacheType.dirtyKey}, strconv.Itoa(record.showTimes), strconv.FormatInt(record.lastShowUnix, 10)).Err()
}

func parseShowRecordKey(cacheType showRecordCacheType, key string) (int, int, bool) {
	prefix := cacheType.prefix + ":"
	if !strings.HasPrefix(key, prefix) {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(key, prefix), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	userId, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	itemId, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return userId, itemId, true
}
