import { ipcRenderer } from 'electron';

window.addEventListener('message', ({ data }) => {
  console.log('Got data' + data);
  ipcRenderer.send('postMessage', data);
});
