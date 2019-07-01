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
*/

import { ipcMain } from 'electron';
import * as config from './config.handlers';
import * as sliver from './sliver.handlers';

interface IPCMessage {
  id: number;
  type: string;
  method: string;
  data: string;
}

const IPCHanadlers = {

  // Config Handlers
  'config_list': config.list,

  // Sliver Handlers
  'sliver_sessions': sliver.sessions,

};


ipcMain.on('ipc', (event: any, data: any) => {
  const msg: IPCMessage = JSON.parse(data);
  const result = IPCHanadlers[msg.method](JSON.parse(msg.data));
  event.sender.send('ipc', {
    id: msg.id,
    type: 'response',
    method: msg.method,
    data: JSON.stringify(result)
  });
});
