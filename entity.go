package optimistic

import "time"

type Entity[Value any] struct {
	expiredAt time.Time
	value     Value
}

func NewEntity[Value any](value Value, ttl time.Duration) Entity[Value] {
	var expiredAt time.Time
	if ttl > 0 {
		expiredAt = time.Now().Add(ttl)
	}
	return Entity[Value]{
		expiredAt: expiredAt,
		value:     value,
	}
}

func (e Entity[Value]) BestBofore() time.Time {
	return e.expiredAt
}

func (e Entity[Value]) Value() Value {
	return e.value
}
