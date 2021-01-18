package db

type inmemdb struct {
	// bucket (string) -> key (string) -> value (bytes)
	mockdb map[string]map[string][][]byte
}

// UseInmemDB uses memory, mainly for testing
func UseInmemDB() Interface {
	return &inmemdb{make(map[string]map[string][][]byte)}
}

func (i inmemdb) Release() error {
	// empties the map
	i.mockdb = make(map[string]map[string][][]byte)
	return nil
}

func (i inmemdb) Dir() string {
	return ""
}

func (i inmemdb) SetDir(dir string) {}

func (i inmemdb) GetAll(bucket string) (entries []*Entry, err error) {
	for k, v := range i.mockdb[bucket] {
		entries = append(entries, &Entry{
			Key:   k,
			Value: v[0],
		})
	}
	return
}

func (i inmemdb) GetAllKey(bucket string, key string) ([][]byte, error) {
	i.initMap(bucket, key)
	return i.mockdb[bucket][key], nil
}

func (i inmemdb) Put(bucket string, key string, value []byte, ttl uint32) error {
	i.initMap(bucket, key)
	if len(i.mockdb[bucket][key]) == 0 {
		i.mockdb[bucket][key] = append(i.mockdb[bucket][key], value)
	} else {
		i.mockdb[bucket][key][0] = value // hardcode idx 0, see GetAll for corresponding retrieval method
	}
	return nil
}

func (i inmemdb) Push(bucket, key string, value []byte) error {
	i.initMap(bucket, key)
	i.mockdb[bucket][key] = append(i.mockdb[bucket][key], value)
	return nil
}

func (i inmemdb) Rem(bucket string, key string, count int) (err error) {
	i.mockdb[bucket][key] = i.mockdb[bucket][key][count:]
	return nil
}

func (i inmemdb) Size(bucket string, key string) (size int, err error) {
	i.initMap(bucket, key)
	return len(i.mockdb[bucket][key]), nil
}

func (i inmemdb) Type() string {
	return "inmem"
}

func (i inmemdb) Truncate(bucket string, key string) error {
	i.initMap(bucket, key)
	delete(i.mockdb[bucket], key)
	return nil
}

func (i inmemdb) initMap(bucket string, key string) {
	if i.mockdb[bucket] == nil {
		i.mockdb[bucket] = make(map[string][][]byte)
	}
	if i.mockdb[bucket][key] == nil {
		i.mockdb[bucket][key] = [][]byte{}
	}
}
