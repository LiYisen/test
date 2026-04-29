package yinyang

type YinYangElement struct {
	High    float64 `json:"high"`
	Low     float64 `json:"low"`
	IsValid bool    `json:"is_valid"`
}

type YinYangState struct {
	IsYang bool           `json:"is_yang"`
	Yang1  YinYangElement `json:"yang1"`
	Yin1   YinYangElement `json:"yin1"`
	Yang2  YinYangElement `json:"yang2"`
	Yin2   YinYangElement `json:"yin2"`
}
