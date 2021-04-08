/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2021 WireGuard LLC. All Rights Reserved.
 */

package device

func (device *Device) DisableSomeRoamingForBrokenMobileSemantics() {
	device.peers.RLock()
	for _, peer := range device.peers.keyMap {
		peer.Lock()
		defer peer.Unlock()
		peer.disableRoaming = peer.endpoint != nil
	}
	device.peers.RUnlock()
}
