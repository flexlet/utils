package utils

import "encoding/json"

type HexBitmap struct {
	Size uint32 `json:"size,omitempty"`
	Bits string `json:"bits,omitempty"`
}

func (b *Bitmap) ToHexBitmap() *HexBitmap {
	return &HexBitmap{
		Size: b.size,
		Bits: b.ToString(),
	}
}

func (h *HexBitmap) ToBitmap() (*Bitmap, error) {
	return FromString(h.Bits, h.Size)
}

func (b *Bitmap) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.ToHexBitmap())
}

func (b *Bitmap) UnmarshalJSON(data []byte) (err error) {
	hb := HexBitmap{}
	if err = json.Unmarshal(data, &hb); err != nil {
		return
	}
	b, err = hb.ToBitmap()
	return
}

func (h *HexBitmap) Set(x uint32) error {
	b, err := h.ToBitmap()
	if err != nil {
		return err
	}
	b.Set(x)
	out := b.ToHexBitmap()
	*h = *out
	return nil
}

func (h *HexBitmap) Remove(x uint32) error {
	b, err := h.ToBitmap()
	if err != nil {
		return err
	}
	b.Remove(x)
	out := b.ToHexBitmap()
	*h = *out
	return nil
}

func (h *HexBitmap) Grow(x uint32) error {
	b, err := h.ToBitmap()
	if err != nil {
		return err
	}
	b.Grow(x)
	out := b.ToHexBitmap()
	*h = *out
	return nil
}
