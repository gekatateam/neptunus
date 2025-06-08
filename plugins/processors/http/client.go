package http

import (
	"net/http"

	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
)

var clientStorage = sharedstorage.New[*http.Client]()
