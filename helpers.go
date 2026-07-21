package caribbeancinemas

import (
	"fmt"
	"net/url"
)

const imgixBase = "https://indy-systems.imgix.net/"

func imgixURL(key string, width int) string {
	if width <= 0 {
		width = 400
	}
	return fmt.Sprintf("%s%s?w=%d&auto=format", imgixBase, url.PathEscape(key), width)
}
