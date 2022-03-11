package d3d

type iD3D11DeviceChildVtbl struct {
	iUnknownVtbl

	GetDevice               uintptr
	GetPrivateData          uintptr
	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
}

type iD3D11DeviceContextVtbl struct {
	iD3D11DeviceChildVtbl

	VSSetConstantBuffers                      uintptr
	PSSetShaderResources                      uintptr
	PSSetShader                               uintptr
	PSSetSamplers                             uintptr
	VSSetShader                               uintptr
	DrawIndexed                               uintptr
	Draw                                      uintptr
	Map                                       uintptr
	Unmap                                     uintptr
	PSSetConstantBuffers                      uintptr
	IASetInputLayout                          uintptr
	IASetVertexBuffers                        uintptr
	IASetIndexBuffer                          uintptr
	DrawIndexedInstanced                      uintptr
	DrawInstanced                             uintptr
	GSSetConstantBuffers                      uintptr
	GSSetShader                               uintptr
	IASetPrimitiveTopology                    uintptr
	VSSetShaderResources                      uintptr
	VSSetSamplers                             uintptr
	Begin                                     uintptr
	End                                       uintptr
	GetData                                   uintptr
	SetPredication                            uintptr
	GSSetShaderResources                      uintptr
	GSSetSamplers                             uintptr
	OMSetRenderTargets                        uintptr
	OMSetRenderTargetsAndUnorderedAccessViews uintptr
	OMSetBlendState                           uintptr
	OMSetDepthStencilState                    uintptr
	SOSetTargets                              uintptr
	DrawAuto                                  uintptr
	DrawIndexedInstancedIndirect              uintptr
	DrawInstancedIndirect                     uintptr
	Dispatch                                  uintptr
	DispatchIndirect                          uintptr
	RSSetState                                uintptr
	RSSetViewports                            uintptr
	RSSetScissorRects                         uintptr
	CopySubresourceRegion                     uintptr
	CopyResource                              uintptr

	/// .....
}

type iD3D11DeviceVtbl struct {
	iUnknownVtbl

	CreateBuffer                         uintptr
	CreateTexture1D                      uintptr
	CreateTexture2D                      uintptr
	CreateTexture3D                      uintptr
	CreateShaderResourceView             uintptr
	CreateUnorderedAccessView            uintptr
	CreateRenderTargetView               uintptr
	CreateDepthStencilView               uintptr
	CreateInputLayout                    uintptr
	CreateVertexShader                   uintptr
	CreateGeometryShader                 uintptr
	CreateGeometryShaderWithStreamOutput uintptr
	CreatePixelShader                    uintptr
	CreateHullShader                     uintptr
	CreateDomainShader                   uintptr
	CreateComputeShader                  uintptr
	CreateClassLinkage                   uintptr
	CreateBlendState                     uintptr
	CreateDepthStencilState              uintptr
	CreateRasterizerState                uintptr
	CreateSamplerState                   uintptr
	CreateQuery                          uintptr
	CreatePredicate                      uintptr
	CreateCounter                        uintptr
	CreateDeferredContext                uintptr
	OpenSharedResource                   uintptr
	CheckFormatSupport                   uintptr
	CheckMultisampleQualityLevels        uintptr
	CheckCounterInfo                     uintptr
	CheckCounter                         uintptr
	CheckFeatureSupport                  uintptr
	GetPrivateData                       uintptr
	SetPrivateData                       uintptr
	SetPrivateDataInterface              uintptr
	GetFeatureLevel                      uintptr
	GetCreationFlags                     uintptr
	GetDeviceRemovedReason               uintptr
	GetImmediateContext                  uintptr
	SetExceptionMode                     uintptr
	GetExceptionMode                     uintptr
}

type iD3D11DebugVtbl struct {
	iUnknownVtbl

	SetFeatureMask             uintptr
	GetFeatureMask             uintptr
	SetPresentPerRenderOpDelay uintptr
	GetPresentPerRenderOpDelay uintptr
	SetSwapChain               uintptr
	GetSwapChain               uintptr
	ValidateContext            uintptr
	ReportLiveDeviceObjects    uintptr
	ValidateContextForDispatch uintptr
}

type iD3D11InfoQueueVtbl struct {
	iUnknownVtbl

	AddApplicationMessage                        uintptr
	AddMessage                                   uintptr
	AddRetrievalFilterEntries                    uintptr
	AddStorageFilterEntries                      uintptr
	ClearRetrievalFilter                         uintptr
	ClearStorageFilter                           uintptr
	ClearStoredMessages                          uintptr
	GetBreakOnCategory                           uintptr
	GetBreakOnID                                 uintptr
	GetBreakOnSeverity                           uintptr
	GetMessage                                   uintptr
	GetMessageCountLimit                         uintptr
	GetMuteDebugOutput                           uintptr
	GetNumMessagesAllowedByStorageFilter         uintptr
	GetNumMessagesDeniedByStorageFilter          uintptr
	GetNumMessagesDiscardedByMessageCountLimit   uintptr
	GetNumStoredMessages                         uintptr
	GetNumStoredMessagesAllowedByRetrievalFilter uintptr
	GetRetrievalFilter                           uintptr
	GetRetrievalFilterStackSize                  uintptr
	GetStorageFilter                             uintptr
	GetStorageFilterStackSize                    uintptr
	PopRetrievalFilter                           uintptr
	PopStorageFilter                             uintptr
	PushCopyOfRetrievalFilter                    uintptr
	PushCopyOfStorageFilter                      uintptr
	PushEmptyRetrievalFilter                     uintptr
	PushEmptyStorageFilter                       uintptr
	PushRetrievalFilter                          uintptr
	PushStorageFilter                            uintptr
	SetBreakOnCategory                           uintptr
	SetBreakOnID                                 uintptr
	SetBreakOnSeverity                           uintptr
	SetMessageCountLimit                         uintptr
	SetMuteDebugOutput                           uintptr
}
type iD3D11ResourceVtbl struct {
	iD3D11DeviceChildVtbl

	GetType             uintptr
	SetEvictionPriority uintptr
	GetEvictionPriority uintptr
}

type iD3D11Texture2DVtbl struct {
	iD3D11ResourceVtbl

	GetDesc uintptr
}
