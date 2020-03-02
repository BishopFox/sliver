

alert(1);


var Sliver = {

  randomId: () => {
    const buf = new Uint32Array(1);
    window.crypto.getRandomValues(buf);
    const bufView = new DataView(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength));
    return bufView.getUint32(0, true) || 1; // In the unlikely event we get a 0 value, return 1 instead
  },

  request: (method, data) => {
    return new Promise((resolve, reject) => {
      const msgId = this.randomId();
      const subscription = IPCResponse$.subscribe((msg) => {
        if (msg.id === msgId) {
          subscription.unsubscribe();
          if (msg.method !== 'error') {
            resolve(msg.data);
          } else {
            reject(msg.data);
          }
        }
      });
      window.postMessage(JSON.stringify({
        id: msgId,
        type: 'request',
        method: method,
        data: data,
      }), '*');
    });
  },

  sessions: async function () {

  }

}

window.addEventListener('message', (ipcEvent) => {
  try {
    const msg = JSON.parse(ipcEvent.data);
    if (msg.type === 'response') {
      IPCResponse$.next(msg);
    } else if (msg.type === 'push') {
      const envelope = pb.Envelope.deserializeBinary(this.decode(msg.data));
      switch (envelope.getType()) {
        case pb.ClientPB.MsgEvent:
          const event = pb.Event.deserializeBinary(envelope.getData_asU8());
          Event$.next(event);
          break;
        case pb.SliverPB.MsgTunnelData:
          const data = pb.TunnelData.deserializeBinary(envelope.getData_asU8());
          TunnelData$.next(data);
          break;
        case pb.SliverPB.MsgTunnelClose:
          const tunCtrl = pb.TunnelClose.deserializeBinary(envelope.getData_asU8());
          TunnelCtrl$.next(tunCtrl);
          break;
        default:
          console.error(`[IPCSciptService] Unknown envelope type ${envelope.getType()}`);
      }
    }
  } catch (err) {
    console.error(`[IPCSciptService] ${err}`);
  }
});