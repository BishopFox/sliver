const { ipcRenderer } = require('electron');

window.addEventListener('message', ({ data }) => {
  console.log('Got data' + data);
  ipcRenderer.send('postMessage', data);
});
