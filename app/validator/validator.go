package validator

import (
	"strings"

	"github.com/lbryio/lbry.go/extras/util"
	v "github.com/lbryio/ozzo-validation"
)

var (
	possibleMediaTypes = []string{"audio", "video", "text", "application", "image", "cad", ""}
	// ClaimTypeValidator is used to validate the claim type parameter
	ClaimTypeValidator = v.NewStringRule(func(str string) bool {
		return util.InSlice(str, []string{"channel", "file"})
	}, "invalid claim type, can only be channel or file")
	// MediaTypeValidator is used to validate the media type parameter
	MediaTypeValidator = v.NewStringRule(func(str string) bool {
		values := strings.Split(str, ",")
		for _, v := range values {
			if !util.InSlice(v, possibleMediaTypes) {
				return false
			}
		}
		return true
	}, "invalid claim type, can only be "+strings.Join(possibleMediaTypes, ","))
)
