package controller

import (
	"net/http/httptest"
	"testing"
	"time"

	"just-vpn/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Exercise each existing payment writer: snapshots must describe the membership
// before this payment, and repeated callbacks must not extend it a second time.
func TestPaymentRenewalSnapshotsAndIdempotency(t *testing.T) {
	for _, provider := range []string{"xx", "ss", "apple"} {
		for _, scenario := range []string{"first_payment", "renewal", "already_paid"} {
			t.Run(provider+"/"+scenario, func(t *testing.T) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = db.Close() })
				orm, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
				if err != nil {
					t.Fatal(err)
				}
				old := model.DB
				model.DB = orm
				t.Cleanup(func() { model.DB = old })
				before := time.Now().In(time.Local).Add(48 * time.Hour).Truncate(time.Second)
				seconds := time.Hour
				payType, paySource := model.OrderPayTypeXXPay, model.OrderPaySourceXXCallback
				if provider == "ss" {
					payType, paySource = model.OrderPayTypeSSPay, model.OrderPaySourceSSCallback
				}
				if provider == "apple" {
					seconds = 24 * time.Hour
					payType, paySource = model.OrderPayTypeAppleIAP, model.OrderPaySourceAppleS2S
				}
				after := before.Add(seconds)
				status := model.OrderPayStatusUnpaid
				if scenario == "already_paid" {
					status = model.OrderPayStatusPaid
				}
				orderRows := func() *sqlmock.Rows {
					return sqlmock.NewRows([]string{"id", "uid", "order_no", "pay_type", "pay_status", "money", "pak_time", "pak_id", "pay_product_id"}).
						AddRow(1, 7, "O1", payType, status, 100, int(seconds/time.Second), 2, "com.example.product")
				}
				userRows := func() *sqlmock.Rows {
					return sqlmock.NewRows([]string{"id", "status", "type", "vip_time"}).AddRow(7, 1, 1, before)
				}
				mock.ExpectBegin()
				if provider == "apple" {
					mock.ExpectQuery("SELECT GET_LOCK").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(1))
					mock.ExpectQuery("SELECT \\* FROM `user`").WithArgs(7, 1).WillReturnRows(userRows())
					existing := sqlmock.NewRows([]string{"id", "uid", "order_no", "pay_status"})
					if scenario == "already_paid" {
						existing.AddRow(1, 7, "O1", model.OrderPayStatusPaid)
					}
					mock.ExpectQuery("SELECT \\* FROM `order` WHERE pay_sn").WithArgs("txn1", 3, 4, 1).WillReturnRows(existing)
					if scenario != "already_paid" {
						mock.ExpectQuery("SELECT \\* FROM `order` WHERE pay_type").WithArgs(payType, "token1", 1).WillReturnRows(orderRows())
					}
				} else {
					mock.ExpectQuery("SELECT \\* FROM `order`").WithArgs("O1", 1).WillReturnRows(orderRows())
					if scenario != "already_paid" {
						mock.ExpectQuery("SELECT \\* FROM `user`").WithArgs(7, 1, 1).WillReturnRows(userRows())
					}
				}
				if scenario != "already_paid" {
					wasPaid := scenario == "renewal"
					mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM `order` WHERE uid = \\? AND pay_status = \\?\\)").
						WithArgs(7, model.OrderPayStatusPaid).WillReturnRows(sqlmock.NewRows([]string{"paid"}).AddRow(wasPaid))
					if provider == "apple" {
						mock.ExpectExec("UPDATE `order` SET `pay_sn`=\\?,`pay_source`=\\?,`pay_status`=\\?,`pay_time`=\\?,`renewal_recorded`=\\?,`renewal_was_paid`=\\?,`verify_time`=\\?,`vip_after`=\\?,`vip_before`=\\?,`update_time`=\\? WHERE id = \\?").
							WithArgs("txn1", paySource, 3, sqlmock.AnyArg(), true, wasPaid, sqlmock.AnyArg(), after, before, sqlmock.AnyArg(), 1).
							WillReturnResult(sqlmock.NewResult(0, 1))
					} else {
						mock.ExpectExec("UPDATE `order` SET `money`=\\?,`pay_sn`=\\?,`pay_source`=\\?,`pay_status`=\\?,`pay_time`=\\?,`renewal_recorded`=\\?,`renewal_was_paid`=\\?,`verify_time`=\\?,`vip_after`=\\?,`vip_before`=\\?,`update_time`=\\? WHERE id = \\?").
							WithArgs(100, "txn1", paySource, 3, sqlmock.AnyArg(), true, wasPaid, sqlmock.AnyArg(), after, before, sqlmock.AnyArg(), 1).
							WillReturnResult(sqlmock.NewResult(0, 1))
					}
					mock.ExpectExec("UPDATE `user` SET `is_pay`=\\?,`pay_all`=pay_all \\+ \\?,`pay_times`=pay_times \\+ \\?,`vip_time`=\\?,`update_time`=\\? WHERE id = \\?").
						WithArgs(1, float64(1), 1, after, sqlmock.AnyArg(), 7).WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectExec("INSERT INTO `user_notice`").WillReturnResult(sqlmock.NewResult(1, 1))
				}
				if provider == "apple" {
					mock.ExpectExec("SELECT RELEASE_LOCK").WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
				}
				mock.ExpectCommit()
				if provider == "apple" {
					var orderNo string
					var vipTime *time.Time
					c, _ := gin.CreateTestContext(httptest.NewRecorder())
					err = orm.Transaction(func(tx *gorm.DB) error {
						return completeAppleOrder(tx, c, 7, model.Package{BaseModel: model.BaseModel{Id: 2}, AppleId: "com.example.product", Day: 1, Price: 100},
							appleTransactionInfo{TransactionId: "txn1", AppAccountToken: "token1"}, paySource, &orderNo, &vipTime)
					})
					wantTime := after
					if scenario == "already_paid" {
						wantTime = before
					}
					if err == nil && (orderNo != "O1" || vipTime == nil || !vipTime.Equal(wantTime)) {
						t.Fatalf("incorrect Apple fulfillment result: %q %v", orderNo, vipTime)
					}
				} else if provider == "ss" {
					err = completeSSPayOrder(map[string]string{"out_trade_no": "O1", "money": "1.00", "trade_no": "txn1"})
				} else {
					err = completeXXPayOrder(map[string]string{"mchOrderNo": "O1", "amount": "100", "payOrderId": "txn1"})
				}
				if err != nil {
					t.Fatal(err)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
