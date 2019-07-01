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
  console.log('ipc send:');
  console.log(event);
  try {
    const msg = JSON.parse(event.data);
    if (msg.type === 'request') {
      ipcRenderer.send('ipc', event.data);
    }
  } catch (err) {
    console.error(err);
  }
});

ipcRenderer.on('ipc', (event, data) => {
  console.log('ipc recv:');
  console.log(event);
  try {
    const msg = JSON.parse(data);
    if (msg.type === 'response') {
      window.postMessage(data, '*');
    }
  } catch (err) {
    console.error(err);
  }
});
