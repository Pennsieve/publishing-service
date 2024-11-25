package utils

const HeaderAuthorization = "Authorization"
const HeaderContentType = "Content-Type"
const HeaderAccept = "Accept"

func StandardResponseHeaders(existing *map[string]string) map[string]string {
	headers := make(map[string]string)
	// copy in any existing headers
	if existing != nil {
		for key, value := range *existing {
			headers[key] = value
		}
	}
	// and add the set of standard response headers
	headers[HeaderContentType] = "application/json; charset=utf-8"
	return headers
}
