package config

type Imap map[interface{}]interface{}

func (s Imap) GetIn(keys ...interface{}) interface{} {
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
	case Imap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return ImapArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}
}

type ImapArray []interface{}

func (s ImapArray) GetIn(keys ...interface{}) interface{} {
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
	case Imap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return ImapArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}

}
