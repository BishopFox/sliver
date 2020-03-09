/*
  Sliver Implant Framework
  Copyright (C) 2019  Bishop Fox
  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.
  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.
  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
-------------------------------------------------------------------------

This preload script is callable from within the sandbox via postMessage,
but should not be directly accessible since it itself is not sandboxed.

*/

const { ipcRenderer } = require('electron');


window.addEventListener('message', (event) => {
  try {
    const msg = JSON.parse(event.data);
    if (msg.type === 'request') {
      if (['client_', 'config_', 'rpc_'].some(prefix => msg.method.startsWith(prefix))) {
        ipcRenderer.send('ipc', msg);
      }
    }
  } catch (err) {
    console.error(err);
  }
});

ipcRenderer.on('ipc', (_, msg) => {
  try {
    if (msg.type === 'response' || msg.type === 'push') {
      window.postMessage(JSON.stringify(msg), '*');
    }
  } catch (err) {
    console.error(err);
  }
});
