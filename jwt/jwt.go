/*
Test JWT encoding/decoding
*/

package jwt

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
)

/*
Encode some scalar types into valid json.
TODO:
1.  Errors.  Want to keep the signature in line with the encoding package.
2.  Shame about the double reflection.
*/
func EncodeToJson(pairs *map[string]interface{}) (js string, err error) {

	for key, val := range *pairs {
		switch reflect.TypeOf(val).String() {
		case "string", "int", "int64", "bool":
			js += fmt.Sprintf("\"%s\":%#v,", key, val)
		default:
			err = fmt.Errorf("BUG: unhandled type: %s\n", reflect.TypeOf(val))
		}
	}

	js = strings.TrimSuffix(js, ",")
	js = "{" + js + "}"
	return js, nil
}

func EncodeToJwt(key, header, payload string) (jwt string, err error) {
	ehed := base64.StdEncoding.EncodeToString([]byte(header))
	epay := base64.StdEncoding.EncodeToString([]byte(payload))
	sign := signature(key, ehed, epay)

	return ehed + "." + epay + "." + sign, nil
}

func VerifyJwt(key, jwt string) (head, payl string, err error) {
	var (
		data []byte
		sign string
	)

	elems := strings.Split(jwt, ".")
	if len(elems) != 3 {
		err = fmt.Errorf("unable to split")
		goto out
	}

	sign = signature(key, elems[0], elems[1])

	if elems[2] != sign {
		err = fmt.Errorf("checksum failure")
		goto out
	}

	if data, err = base64.StdEncoding.DecodeString(elems[0]); err != nil {
		goto out
	}
	head = string(data)

	if data, err = base64.StdEncoding.DecodeString(elems[1]); err != nil {
		goto out
	}
	payl = string(data)

out:
	return
}

func signature(key, head, payl string) (sign string) {

	hh := sha256.New()
	hh.Write([]byte(key + head + "." + payl))
	summ := fmt.Sprintf("%x", hh.Sum(nil))
	sign = base64.StdEncoding.EncodeToString([]byte(summ))

	return
}