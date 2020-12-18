package syncer

import (
	"testing"
)

func BenchmarkStoreInsert(b *testing.B) {
	cfg := DefaultStoreConfig
	cfg.dbConfig.Dir = "./test"
	cfg.interval = 1
	cfg.models = map[string]interface{}{
		"SyncerTest": &syncerTest{},
		"StoreTest":  &storeTest{},
	}
	store := NewStore(cfg)
	setUpStore(store)
	defer tearDownStore(store)

	model := &storeTest{1, []byte(`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus sit amet tellus vitae neque varius ultrices nec nec sem. Vivamus finibus orci nec erat blandit ultricies. Ut non nisl magna. Nunc vel imperdiet quam, ut dignissim diam. Suspendisse in augue ex. Morbi ullamcorper placerat erat id volutpat. Duis lobortis urna non nibh efficitur viverra. Nulla facilisi. Donec maximus eu nunc vel pharetra. Proin sit amet orci tincidunt, bibendum lacus et, mattis nisl. Etiam eget nisl a nisi euismod consequat. In ut maximus lorem.
	Aenean pharetra quis urna at condimentum. Aenean et convallis justo, quis tristique sem. Pellentesque placerat eleifend purus. Fusce interdum sagittis auctor. Suspendisse rutrum magna ultrices, egestas massa non, interdum dolor. Nulla rhoncus laoreet tellus et mollis. Nulla pulvinar faucibus purus, non ultrices odio luctus in. Ut id accumsan eros, ut volutpat ex. Pellentesque non enim dignissim, consequat tortor vel, iaculis dolor. Sed at nisl eros. Donec vel dapibus ex. Proin erat odio, pretium in porttitor ac, feugiat nec nisl. Nunc hendrerit enim eget ipsum sodales, porttitor aliquam urna molestie.
	Nulla suscipit tellus et elit ornare iaculis. Nullam vel nulla mollis, sodales diam eu, cursus ipsum. Cras at felis non neque bibendum semper sed a eros. Sed non porta nulla. Phasellus vehicula, lacus id laoreet convallis, sem purus ultricies sapien, vitae porta nisi lacus in libero. Aenean eget sem malesuada, tincidunt sem eu, sodales risus. Mauris sapien arcu, fermentum a blandit et, fringilla ut est. Praesent in massa ut ipsum imperdiet vestibulum. Integer at fermentum tellus. Donec quis turpis massa. Ut eget rhoncus est. Sed lacinia, ex et dignissim porta, felis lectus hendrerit mi, et tempus eros justo id quam. Nullam volutpat erat lacus, non consectetur nunc bibendum sit amet.
	Aenean vehicula elit a leo sodales congue. Nulla eu augue sed turpis ullamcorper blandit nec eget mi. Nam dui urna, interdum et nibh non, egestas placerat magna. Sed in sed.`)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := store.LogInsert("StoreTest", model)
		if err != nil {
			b.Errorf("Insert storeTest failed: %v\n", err.Error())
			b.FailNow()
		}
	}
}

func BenchmarkStoreRead(b *testing.B) {
	b.StopTimer()
	cfg := DefaultStoreConfig
	cfg.dbConfig.Dir = "./test"
	cfg.models = map[string]interface{}{
		"SyncerTest": &syncerTest{},
		"StoreTest":  &storeTest{},
	}
	store := NewStore(cfg)
	setUpStore(store)
	defer tearDownStore(store)

	model := &storeTest{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		store.GetAll("StoreTest", model, func(rec *Record) bool {
			return true
		})
	}
}
