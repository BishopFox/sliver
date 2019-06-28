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

import { app, ipcMain, BrowserWindow, Menu, screen } from 'electron';
import { RPCClient, RPCConfig } from './rpc';
import * as fs from 'fs';
import * as path from 'path';
import * as url from 'url';
import * as pb from './pb/sliver_pb';
import { ClientPB, SliverPB } from './pb/constants';

let mainWindow, serve;
const args = process.argv.slice(1);
serve = args.some(val => val === '--serve');

async function createMainWindow() {

  const electronScreen = screen;
  const size = electronScreen.getPrimaryDisplay().workAreaSize;

  // Create the browser window.
  mainWindow = new BrowserWindow({
    x: 0,
    y: 0,
    // width: size.width,
    // height: size.height,
    width: 800,
    height: 600,
    webPreferences: {
      nodeIntegration: false,
      preload: path.join(__dirname, 'preload.js')
    },
  });

  if (serve) {
    require('electron-reload')(__dirname, {
      electron: require(`${__dirname}/node_modules/electron`)
    });
    mainWindow.loadURL('http://localhost:4200');
  } else {
    mainWindow.loadURL(url.format({
      pathname: path.join(__dirname, 'dist/index.html'),
      protocol: 'file:',
      slashes: true
    }));
  }

  if (serve) {
    mainWindow.webContents.openDevTools();
  }


  // Emitted when the window is closed.
  mainWindow.on('closed', () => {
    // Dereference the window object, usually you would store window
    // in an array if your app supports multi windows, this is the time
    // when you should delete the corresponding element.
    mainWindow = null;
  });

  console.log('Main window done');
}


try {

  console.log('Loading client config ...');
  const rawConfig = fs.readFileSync('/Users/moloch/.sliver-client/configs/moloch_lil-peep.rip.cfg');
  const config: RPCConfig = JSON.parse(rawConfig.toString('utf8'));
  const rpcClient = new RPCClient(config);
  rpcClient.connect().then((rpc) => {
    rpc.subscribe(envelope => {
      console.log(`Recv envelope: ${envelope}`);
    });

    const sessionsReq = new pb.Envelope();
    sessionsReq.setType(ClientPB.MsgSessions);
    sessionsReq.setTimeout(360);
    rpc.next(sessionsReq);
    console.log('rpc done');

  });

  // This method will be called when Electron has finished
  // initialization and is ready to create browser windows.
  // Some APIs can only be used after this event occurs.
  app.on('ready', createMainWindow);

  // Quit when all windows are closed.
  app.on('window-all-closed', () => {
    // On OS X it is common for applications and their menu bar
    // to stay active until the user quits explicitly with Cmd + Q
    if (process.platform !== 'darwin') {
      app.quit();
    }
  });

  app.on('activate', () => {
    if (mainWindow === null) {
      createMainWindow();
    }
  });

} catch (error) {
  throw error;
}


ipcMain.on('postMessage', (event) => {
  console.log(event);
  mainWindow.webContents.send('hi back');
});
