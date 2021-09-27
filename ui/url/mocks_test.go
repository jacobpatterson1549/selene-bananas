//go:build js && wasm

package url

type mockURIComponentEncoder struct {
	EncodeURIComponentFunc func(str string) string
}

func (m mockURIComponentEncoder) EncodeURIComponent(str string) string {
	return m.EncodeURIComponentFunc(str)
}
