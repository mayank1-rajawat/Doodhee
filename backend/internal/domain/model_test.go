package domain

import "testing"

func TestMoneyAndTransitions(t *testing.T) {
	for _, v := range []struct{ q, p, w string }{{"1.5", "65.00", "97.50"}, {"0.125", "60.00", "7.50"}, {"3", "0.10", "0.30"}} {
		got, e := Total(v.q, v.p)
		if e != nil || got != v.w {
			t.Fatal(got, e)
		}
	}
	if _, e := Total("-1", "65"); e == nil {
		t.Fatal("negative quantity")
	}
	if e := Transition("delivered", "accepted", "", false); e == nil {
		t.Fatal("terminal transition")
	}
	if e := Transition("out_for_delivery", "not_delivered", "", false); e == nil {
		t.Fatal("missing reason")
	}
	if e := Transition("out_for_delivery", "not_delivered", "other", false); e != nil {
		t.Fatal(e)
	}
}
