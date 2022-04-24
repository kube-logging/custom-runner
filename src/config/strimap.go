package config

type Strimap map[string]interface{}

func (s Strimap) GetIn(keys ...interface{}) interface{} {
	if len(keys) == 0 {
		return s
	}
	key, subKeys := keys[0], keys[1:]
	v, ok := key.(string)
	if !ok {
		return nil
	}
	subStore, ok := s[v]
	if !ok {
		return nil
	}
	switch v := subStore.(type) {
	case map[string]interface{}:
		return Strimap(v).GetIn(subKeys...)
	case Strimap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return StriArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}
}

type StriArray []interface{}

func (s StriArray) GetIn(keys ...interface{}) interface{} {
	if len(keys) == 0 {
		return []interface{}(s)
	}
	key, subKeys := keys[0], keys[1:]
	intKey, ok := key.(int)
	if !ok {
		return nil
	}

	if len(s) <= intKey || intKey < 0 {
		return nil
	}
	subStore := s[intKey]
	switch v := subStore.(type) {
	case Strimap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return StriArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}

}
