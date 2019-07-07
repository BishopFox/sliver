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

import { app, BrowserWindow, screen } from 'electron';
import { startIPCHandlers } from './ipc';
import * as path from 'path';
import * as url from 'url';

let mainWindow, serve;
const args = process.argv.slice(1);
serve = args.some(val => val === '--serve');

process.on('unhandledRejection', (reason, p) => {
  console.log('Unhandled Rejection at: Promise', p, 'reason:', reason);
});

async function createMainWindow() {

  const electronScreen = screen;
  const size = electronScreen.getPrimaryDisplay().workAreaSize;

  // Create the browser window.
  const gutterSize = 100;
  mainWindow = new BrowserWindow({
    // frame: false,
    titleBarStyle: 'hiddenInset',
    x: gutterSize,
    y: gutterSize,
    width: size.width - gutterSize,
    height: size.height - gutterSize,
    webPreferences: {
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
      preload: path.join(__dirname, 'preload.js'),
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

// --------------------------------------------- [ MAIN ] ---------------------------------------------

try {
  // This method will be called when Electron has finished
  // initialization and is ready to create browser windows.
  // Some APIs can only be used after this event occurs.
  app.on('ready', () => {
    createMainWindow();
    startIPCHandlers(mainWindow);
  });

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
