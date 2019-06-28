"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : new P(function (resolve) { resolve(result.value); }).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
var rxjs_1 = require("rxjs");
var tls_1 = require("tls");
var pb = require("./pb/sliver_pb");
var RPCClient = /** @class */ (function () {
    function RPCClient(config) {
        this.isConnected = false;
        this.config = config;
    }
    // This method returns a Subject that shits out
    // or takes in pb.Envelopes and abstracts the byte
    // non-sense for your.
    RPCClient.prototype.connect = function () {
        return __awaiter(this, void 0, void 0, function () {
            var _this = this;
            return __generator(this, function (_a) {
                return [2 /*return*/, new Promise(function (resolve, reject) { return __awaiter(_this, void 0, void 0, function () {
                        var tlsSubject, envelopeObservable, envelopeObserver;
                        var _this = this;
                        return __generator(this, function (_a) {
                            switch (_a.label) {
                                case 0:
                                    if (this.isConnected) {
                                        reject('Already connected to rpc server');
                                    }
                                    return [4 /*yield*/, this.tlsConnect()];
                                case 1:
                                    tlsSubject = _a.sent();
                                    this.isConnected = true;
                                    envelopeObservable = rxjs_1.Observable.create(function (obs) {
                                        _this.recvBuffer = Buffer.alloc(0);
                                        tlsSubject.subscribe(function (recvData) {
                                            console.log("Read " + recvData.length + " bytes");
                                            _this.recvBuffer = Buffer.concat([_this.recvBuffer, recvData]);
                                            if (4 <= _this.recvBuffer.length) {
                                                var readSize = new Int32Array(_this.recvBuffer.slice(0, 4))[0];
                                                console.log("Recv msg length: " + readSize + " bytes");
                                                if (readSize <= 4 + _this.recvBuffer.length) {
                                                    console.log('Parsing out message from recvBuffer');
                                                    var bytes = _this.recvBuffer.slice(4, 4 + readSize);
                                                    var envelope = pb.Envelope.deserializeBinary(bytes);
                                                    _this.recvBuffer = Buffer.from(_this.recvBuffer.slice(4 + readSize));
                                                    obs.next(envelope);
                                                }
                                            }
                                        });
                                    });
                                    envelopeObserver = {
                                        next: function (envelope) {
                                            var dataBuffer = Buffer.from(envelope.serializeBinary());
                                            var sizeBuffer = _this.toBytesUint32(dataBuffer.length);
                                            console.log("Sending msg (" + envelope.getType() + "): " + dataBuffer.length + " bytes ...");
                                            tlsSubject.next(Buffer.concat([sizeBuffer, dataBuffer]));
                                        }
                                    };
                                    resolve(rxjs_1.Subject.create(envelopeObserver, envelopeObservable));
                                    return [2 /*return*/];
                            }
                        });
                    }); })];
            });
        });
    };
    RPCClient.prototype.toBytesUint32 = function (num) {
        var arr = new ArrayBuffer(4); // an Int32 takes 4 bytes
        var view = new DataView(arr);
        view.setUint32(0, num, false); // byteOffset = 0; litteEndian = false
        return Buffer.from(arr);
    };
    Object.defineProperty(RPCClient.prototype, "tlsOptions", {
        get: function () {
            return {
                ca: this.config.ca_certificate,
                key: this.config.private_key,
                cert: this.config.certificate,
                host: this.config.lhost,
                port: this.config.lport,
                rejectUnauthorized: true,
                // This should ONLY skip verifying the hostname matches the cerftificate:
                // https://nodejs.org/api/tls.html#tls_tls_checkserveridentity_hostname_cert
                checkServerIdentity: function () { return undefined; },
            };
        },
        enumerable: true,
        configurable: true
    });
    // This is somehow the "clean" way to do this shit...
    // tlsConnect returns a Subject that shits out Buffers
    // or takes in Buffers of an interminate size as they come
    RPCClient.prototype.tlsConnect = function () {
        var _this = this;
        return new Promise(function (resolve, reject) {
            console.log("Connecting to " + _this.config.lhost + ":" + _this.config.lport + " ...");
            // Conenct to the server
            _this.socket = tls_1.connect(_this.tlsOptions);
            _this.socket.setNoDelay(true);
            // This event fires after the tls handshake, but we need to check `socket.authorized`
            _this.socket.on('secureConnect', function () {
                console.log('RPC client connected', _this.socket.authorized ? 'authorized' : 'unauthorized');
                if (_this.socket.authorized === true) {
                    var socketObservable = rxjs_1.Observable.create(function (obs) {
                        _this.socket.on('data', function (data) {
                            console.log("Socket read " + data.length + " bytes");
                            obs.next(data);
                        }); // Bind observable's .next() to 'data' event
                        _this.socket.on('close', obs.error.bind(obs)); // same with close/error
                    });
                    var socketObserver = {
                        next: function (data) {
                            console.log("Socket write " + data.length + " bytes");
                            console.log(data.toString('utf8'));
                            _this.socket.write(data, function () {
                                console.log("Socket write completed");
                            });
                        }
                    };
                    resolve(rxjs_1.Subject.create(socketObserver, socketObservable));
                }
                else {
                    reject('Unauthorized connection');
                }
            });
        });
    };
    return RPCClient;
}());
exports.RPCClient = RPCClient;
//# sourceMappingURL=rpc.js.map