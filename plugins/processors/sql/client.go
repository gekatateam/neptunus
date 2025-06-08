package sql

import (
	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
)

var clientStorage = sharedstorage.New[*sqlx.DB]()
