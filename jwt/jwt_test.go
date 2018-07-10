/*
Not so sure the string.Builder is a great idea.
*/
package jwt

import (
	"testing"
)

func setup() (head, payl map[string]interface{}) {
	head = map[string]interface{}{}
	head["alg"] = "HS256"
	head["typ"] = "JWT"
	head["boo"] = false

	payl = map[string]interface{}{}
	payl["loggedInAs"] = "admin"
	payl["iat"] = 1530000000
	payl["boo"] = true

	return
}

func TestEncodeJson(t *testing.T) {

	foo, _ := setup()

	if _, err := EncodeToJson(&foo); err != nil {
		t.Error(err)
	}
}

func TestEncodeJwt(t *testing.T) {

	foo, bar := setup()

	head, _ := EncodeToJson(&foo)
	payl, _ := EncodeToJson(&bar)
	if _, err := EncodeToJwt("123", head, payl); err != nil {
		t.Error(err)
	}
}

func TestDecode(t *testing.T) {

	foo, bar := setup()

	head, _ := EncodeToJson(&foo)
	payl, _ := EncodeToJson(&bar)
	jtok, _ := EncodeToJwt("123", head, payl)

	hed2, pay2, err := VerifyJwt("123", jtok)

	if err != nil {
		t.Error(err)
	}

	if head != hed2 {
		t.Error("failed to decode header")
	}

	if payl != pay2 {
		t.Error("failed to decode payload")
	}
}

/*
Benchmarking shows that a string.Builder makes almost no difference,
and string concat is easier on the brain.

go test -bench=.
goos: linux
goarch: amd64
Benchmark_Type_01-2       100000         16912 ns/op 	using string concat
Benchmark_Type_02-2       100000         16746 ns/op 	using string.Builder
PASS
ok  	_/home/mike/Dev/Openid/jwt	3.737s

func Benchmark_Type_01(b *testing.B) {
	
	foo := map[string]interface{}{}
	bar := map[string]interface{}{}

	foo = map[string]interface{}{}
	foo["alg"] = "HS256"
	foo["typ"] = "JWT"
	foo["boo"] = false

	bar = map[string]interface{}{}
	bar["loggedInAs"] = "admin"
	bar["iat"] = 1530000000
	bar["boo"] = true

	for nn := 0; nn < b.N; nn++ {
		head, _ := EncodeToJson_01(&foo)
		payl, _ := EncodeToJson_01(&bar)
		EncodeToJwt("123", head, payl)
	}
}

func Benchmark_Type_02(b *testing.B) {
	
	foo := map[string]interface{}{}
	bar := map[string]interface{}{}

	foo = map[string]interface{}{}
	foo["alg"] = "HS256"
	foo["typ"] = "JWT"
	foo["boo"] = false

	bar = map[string]interface{}{}
	bar["loggedInAs"] = "admin"
	bar["iat"] = 1530000000
	bar["boo"] = true

	for nn := 0; nn < b.N; nn++ {
		head, _ := EncodeToJson_02(&foo)
		payl, _ := EncodeToJson_02(&bar)
		EncodeToJwt("123", head, payl)
	}
}
*/
