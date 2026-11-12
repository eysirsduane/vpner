package model

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func TestRenewalCoverageDoesNotInventHistoricalData(t *testing.T) {
	loc := time.FixedZone("CN", 8*3600)
	since := time.Date(2026, 9, 13, 15, 0, 0, 0, loc)
	for _, tc := range []struct {
		day           int
		daily, weekly bool
	}{{13, false, false}, {14, true, false}, {19, true, false}, {20, true, true}} {
		d, w := renewalCoverage(time.Date(2026, 9, tc.day, 0, 0, 0, 0, loc), since)
		if d != tc.daily || w != tc.weekly {
			t.Fatalf("coverage day %d: %v %v", tc.day, d, w)
		}
	}
}

func TestRenewalSnapshotFieldsDoNotChangeClientOrderJSON(t *testing.T) {
	now := time.Now()
	encoded, err := json.Marshal(Order{RenewalOrderSnapshot: RenewalOrderSnapshot{RenewalRecorded: true, RenewalWasPaid: true, VipBefore: &now, VipAfter: &now}})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"renewal_recorded", "renewal_was_paid", "vip_before", "vip_after"} {
		if bytes.Contains(encoded, []byte(key)) {
			t.Fatalf("internal field exposed: %s", key)
		}
	}
}

func TestRenewalDoesNotInstallUserUpdateHook(t *testing.T) {
	if _, exists := reflect.TypeOf(&User{}).MethodByName("BeforeUpdate"); exists {
		t.Fatal("renewal statistics must not intercept ordinary user updates")
	}
}

func TestRenewalCohortSQLCountsUsersAfterExtensions(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 sqlite fixture required")
	}
	script := `import sqlite3,sys,json
db=sqlite3.connect(':memory:')
db.execute('CREATE TABLE user(id INTEGER,is_pay INTEGER,pay_times INTEGER,vip_time TEXT)')
db.execute('CREATE TABLE "order"(uid INTEGER,pay_status INTEGER,renewal_recorded INTEGER,renewal_was_paid INTEGER,vip_before TEXT,pay_time TEXT)')
users=[(1,0,0,'2026-09-13 10:00:00'),(2,1,1,'2026-10-13 10:00:00'),(3,1,1,'2026-09-09 10:00:00'),(4,1,1,'2026-10-12 10:00:00'),(5,1,1,'2026-10-13 10:00:00'),(6,1,2,'2026-11-13 10:00:00'),(7,1,2,'2026-10-13 10:00:00'),(8,1,1,'2026-10-13 10:00:00'),(9,1,1,'2026-10-13 10:00:00'),(10,1,2,'2026-10-13 10:00:00'),(11,1,2,'2026-10-20 10:00:00'),(12,1,1,'2026-09-13 23:00:00')]
db.executemany('INSERT INTO user VALUES(?,?,?,?)',users)
orders=[(2,3,1,1,'2026-09-13 20:00:00','2026-09-13 09:00:00'),(4,3,1,1,'2026-09-10 10:00:00','2026-09-12 10:00:00'),(5,3,1,1,'2026-09-14 10:00:00','2026-09-13 10:00:00'),(6,3,1,1,'2026-09-12 10:00:00','2026-09-13 10:00:00'),(6,3,1,1,'2026-10-13 10:00:00','2026-09-13 11:00:00'),(7,3,1,1,'2026-09-13 10:00:00','2026-09-12 10:00:00'),(8,3,1,0,'2026-09-13 10:00:00','2026-09-13 10:00:00'),(9,4,1,1,'2026-09-13 10:00:00','2026-09-13 10:00:00'),(10,3,1,1,'2026-09-06 10:00:00','2026-09-13 10:00:00'),(11,3,1,1,'2026-09-09 10:00:00','2026-09-20 10:00:00')]
db.executemany('INSERT INTO "order" VALUES(?,?,?,?,?,?)',orders)
db.executemany('INSERT INTO "order" VALUES(?,?,?,?,?,?)',[(uid,3,1,0,None,'2026-07-01 00:00:00') for uid in range(2,15) if uid!=8])
params=['2026-09-13 00:00:00']*2+['2026-09-07 00:00:00','2026-09-14 00:00:00']*2+['2026-09-13 12:00:00']+['2026-09-07 00:00:00','2026-09-14 00:00:00']
row=db.execute(sys.argv[1],params).fetchone()
assert row==(4,2,8,4), row
# User 2 expires at 20:00 but paid at 09:00: same-date renewal counts.
# User 4 expired within seven days and already renewed on Sep 12. A further
# advance purchase today adds nothing; Sep 12 renewal already counts weekly.
db.execute('INSERT INTO "order" VALUES(?,?,?,?,?,?)',(4,3,1,1,'2026-10-12 10:00:00','2026-09-13 09:00:00'))
db.execute('UPDATE user SET vip_time=? WHERE id=4',('2026-11-12 10:00:00',))
row=db.execute(sys.argv[1],params).fetchone()
assert row==(4,2,8,4), row
# Additional repeat purchases today must not count the same user twice.
db.execute('INSERT INTO "order" VALUES(?,?,?,?,?,?)',(4,3,1,1,'2026-11-12 10:00:00','2026-09-13 10:00:00'))
db.execute('UPDATE user SET vip_time=? WHERE id=4',('2026-12-12 10:00:00',))
assert db.execute(sys.argv[1],params).fetchone()==(4,2,8,4)
# Seven-day boundary: Sep 7 qualifies; Sep 6 does not (user 10 above).
db.execute('INSERT INTO user VALUES(16,1,2,?)',('2026-10-13 10:00:00',))
db.execute('INSERT INTO "order" VALUES(?,?,?,?,?,?)',(16,3,1,1,'2026-09-07 00:00:00','2026-09-13 10:00:00'))
assert db.execute(sys.argv[1],params).fetchone()==(4,2,9,5)
# Earlier payments in the seven-day window count, including multiple payments
# by the same user, without changing today's numerator.
db.execute('INSERT INTO "order" VALUES(?,?,?,?,?,?)',(16,3,1,1,'2026-09-07 00:00:00','2026-09-07 00:00:00'))
assert db.execute(sys.argv[1],params).fetchone()==(4,2,9,5)
# Refunds are not successful renewals; future payments do not count yet.
# Removing Sep 12's successful renewal drops only the weekly numerator.
db.execute('UPDATE "order" SET pay_status=4 WHERE uid=4 AND vip_before=?',('2026-09-10 10:00:00',))
assert db.execute(sys.argv[1],params).fetchone()==(4,2,9,4)
# Multiple qualifying orders for one user still count that user once.
db.execute('INSERT INTO "order" VALUES(?,?,?,?,?,?)',(16,3,1,1,'2026-09-07 00:00:00','2026-09-13 11:00:00'))
assert db.execute(sys.argv[1],params).fetchone()==(4,2,9,4)
# A payment before the entire expiry window still counts for today's cohort.
db.execute('INSERT INTO user VALUES(17,1,2,?)',('2026-10-13 10:00:00',))
db.execute('INSERT INTO "order" VALUES(?,?,?,?,?,?)',(17,3,1,1,'2026-09-13 10:00:00','2026-09-01 10:00:00'))
assert db.execute(sys.argv[1],params).fetchone()==(5,3,10,5)
# A payment at the report cutoff is not yet included.
db.execute('INSERT INTO "order" VALUES(?,?,?,?,?,?)',(12,3,1,1,'2026-09-13 23:00:00','2026-09-13 12:00:00'))
assert db.execute(sys.argv[1],params).fetchone()==(5,3,10,5)
print(json.dumps(row))
`
	output, err := exec.Command(python, "-I", "-S", "-c", script, renewalCohortSQL).CombinedOutput()
	if err != nil {
		t.Fatalf("cohort SQL fixture: %v %s", err, output)
	}
}
