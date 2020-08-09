// +build js,wasm

package user

import (
	"errors"
	"syscall/js"

	"github.com/jacobpatterson1549/selene-bananas/ui/dom/json"
)

// parseUserInfoJSON converts the text into a user info.
func parseUserInfoJSON(text string) (*userInfo, error) {
	claims, err := json.Parse(text)
	switch {
	case err != nil:
		return nil, errors.New("parsing user info json: " + err.Error())
	case claims.Type() != js.TypeObject:
		return nil, errors.New("wanted user info to be an object, got " + claims.Type().String())
	case claims.Get("sub").IsUndefined():
		return nil, errors.New("no 'sub' field in user claims")
	case claims.Get("sub").Type() != js.TypeString:
		return nil, errors.New("sub is not a string")
	case claims.Get("points").IsUndefined():
		return nil, errors.New("no 'points' field in user claims")
	case claims.Get("points").Type() != js.TypeNumber:
		return nil, errors.New("points is not a number")
	}
	u := userInfo{
		username: claims.Get("sub").String(),
		points:   claims.Get("points").Int(),
	}
	return &u, nil
}
