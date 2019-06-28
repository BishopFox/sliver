import { app, ipcMain, BrowserWindow, Menu, screen } from 'electron';
import { RPCClient, RPCConfig } from './rpc';
import * as fs from 'fs';
import * as path from 'path';
import * as url from 'url';


let mainWindow, serve;
const args = process.argv.slice(1);
serve = args.some(val => val === '--serve');

async function createMainWindow() {


  console.log('Loading client config ...');
  const rawConfig = fs.readFileSync('/Users/moloch/.sliver-client/configs/moloch_lil-peep.rip.cfg');
  const config: RPCConfig = JSON.parse(rawConfig.toString('utf8'));
  const rpc = new RPCClient(config);
  await rpc.connect();

  const electronScreen = screen;
  const size = electronScreen.getPrimaryDisplay().workAreaSize;

  // Create the browser window.
  mainWindow = new BrowserWindow({
    x: 0,
    y: 0,
    width: size.width,
    height: size.height,
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

}


try {

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
