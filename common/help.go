package common

import (
	d "github.com/lehotomi/diam/diam"
)

func Result_code_to_string(rc int) string {

	switch rc {
	case d.SUCCESS:
		return "SUCCESS"
	case d.END_USER_SERVICE_DENIED:
		return "END_USER_SERVICE_DENIED"
	case d.CREDIT_CONTROL_NOT_APPLICABLE:
		return "CREDIT_CONTROL_NOT_APPLICABLE"
	case d.CREDIT_LIMIT_REACHED:
		return "CREDIT_LIMIT_REACHED"
	case d.USER_UNKNOWN:
		return "USER_UNKNOWN"
	case d.UNKNOWN_SESSION_ID:
		return "UNKNOWN_SESSION_ID"
	case d.RATING_FAILED:
		return "RATING_FAILED"
	case d.AUTHORIZATION_REJECTED:
		return "AUTHORIZATION_REJECTED"
	case d.AUTHENTICATION_REJECTED:
		return "AUTHENTICATION_REJECTED"
	case d.UNABLE_TO_DELIVER:
		return "UNABLE_TO_DELIVER"
	default:
		return "UNKNOWN"
	}

}