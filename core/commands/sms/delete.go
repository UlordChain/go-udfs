package sms

import (
	"fmt"
	"net/http"
)

func Delete(token, hash string) (err error) {
	policy, err := parsePolicy(token)
	if err != nil {
		return
	}

	url := fmt.Sprintf("%s/v%d/resource/%s", SMSAddr, policy.Ver, hash)

	r, err :=http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return
	}
	r.Header.Set("token", token)

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return
	}

	err = handleResp(resp)
	return
}
