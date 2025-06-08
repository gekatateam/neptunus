package deduplicate

import (
	"github.com/redis/go-redis/v9"

	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
)

var clientStorage = sharedstorage.New[redis.UniversalClient]()
