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

import { app, BrowserWindow, screen, protocol } from 'electron';
import * as path from 'path';

import { startIPCHandlers, IPCHandlers } from './ipc/ipc';
import * as AppProtocol from './app-protocol';


let mainWindow: BrowserWindow;


async function createMainWindow() {

  const electronScreen = screen;
  const size = electronScreen.getPrimaryDisplay().workAreaSize;

  // Create the browser window.
  const gutterSize = 100;
  mainWindow = new BrowserWindow({
    titleBarStyle: 'hidden',
    x: gutterSize,
    y: gutterSize,
    width: size.width - (gutterSize * 2),
    height: size.height - (gutterSize * 2),
    webPreferences: {
      scrollBounce: true,
      // I think I got all of the settings we want here to reasonably lock down
      // the BrowserWindow - https://electronjs.org/docs/api/browser-window
      sandbox: true,
      webSecurity: true,
      contextIsolation: true,
      webviewTag: false,
      enableRemoteModule: false,
      allowRunningInsecureContent: false,
      nodeIntegration: false,
      nodeIntegrationInWorker: false,
      nodeIntegrationInSubFrames: false,
      nativeWindowOpen: false,
      safeDialogs: true,
      preload: path.join(__dirname, 'preload.js'),
    },
    show: false, // hide until 'ready-to-show'
  });

  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  mainWindow.loadURL(`${AppProtocol.scheme}://sliver/index.html`);
  mainWindow.webContents.openDevTools();

  // Emitted when the window is closed.
  mainWindow.on('closed', () => {
    // Dereference the window object, usually you would store window
    // in an array if your app supports multi windows, this is the time
    // when you should delete the corresponding element.
    mainWindow = null;
  });

  console.log('Main window done');

}

// ----------------------------------------- [ MAIN ] -----------------------------------------

try {

  // Custom procotol handler
  app.on('ready', () => {
    protocol.registerBufferProtocol(AppProtocol.scheme, AppProtocol.requestHandler, (err) => {
      if (err) {
        console.error(`[app-protocol] Error: ${err}`);
      }
    });
    createMainWindow();
    startIPCHandlers(mainWindow);
  });

  // Prevent navigation in any window
  // WARNING: This actually doesn't work because Electron hates security
  // https://github.com/electron/electron/issues/8841
  app.on('web-contents-created', (_, contents) => {
    contents.on('will-navigate', (event, url) => {
      console.log(`[will-navigate] ${url}`);
      console.log(event);
      event.preventDefault();
    });
    contents.on('will-redirect', (event, url) => {
      console.log(`[will-redirect] ${url}`);
      console.log(event);
      event.preventDefault();
    });
  });

  protocol.registerSchemesAsPrivileged([{
    scheme: AppProtocol.scheme,
    privileges: { standard: true, secure: true }
  }]);

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
  console.log(error);
  process.exit(1);
}
