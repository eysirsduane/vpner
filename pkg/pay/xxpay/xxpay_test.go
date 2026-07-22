package xxpay

import "testing"

func TestSignIgnoresEmptyAndSignField(t *testing.T) {
	params := map[string]string{
		"userId":     "test01",
		"type":       "wechat",
		"money":      "2.0",
		"remark":     "",
		"outTradeNo": "P12312321123",
		"sign":       "ignored",
	}
	got := Sign(params, "EWEFD123RGSRETYDFNGFGFGSHDFGH")
	const want = "5E0AA05DD4BB4FE5AB65608123EBA591"
	if got != want {
		t.Fatalf("Sign() = %s, want %s", got, want)
	}
}

func TestVerifyJSONSignSupportsExtraScalarFields(t *testing.T) {
	params := map[string]string{
		"retCode":     "0",
		"payOrderId":  "P01201907231119090520000",
		"payMethod":   "formJump",
		"payJumpUrl":  "https://pay.example.com/jump",
		"needQuery":   "true",
		"orderStatus": "0",
	}
	params["sign"] = Sign(params, "secret")

	body := []byte(`{"retCode":"0","payOrderId":"P01201907231119090520000","payMethod":"formJump","payJumpUrl":"https://pay.example.com/jump","needQuery":true,"orderStatus":"0","sign":"` + params["sign"] + `"}`)
	if !VerifyJSONSign(body, "secret") {
		t.Fatal("expected json sign to verify")
	}
}
