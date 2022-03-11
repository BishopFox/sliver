package d3d

type _D3D11_BOX struct {
	Left, Top, Front, Right, Bottom, Back uint32
}

type _D3D11_TEXTURE2D_DESC struct {
	Width          uint32
	Height         uint32
	MipLevels      uint32
	ArraySize      uint32
	Format         uint32
	SampleDesc     _DXGI_SAMPLE_DESC
	Usage          uint32
	BindFlags      uint32
	CPUAccessFlags uint32
	MiscFlags      uint32
}
