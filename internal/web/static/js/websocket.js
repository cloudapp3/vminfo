function VMInfoWebSocket(onMessage, onStatus) {
    this.onMessage = onMessage;
    this.onStatus = onStatus;
    this.ws = null;
    this.reconnectDelay = 1000;
    this.maxReconnectDelay = 30000;
    this.currentDelay = this.reconnectDelay;
    this._connect();
}

VMInfoWebSocket.prototype._connect = function() {
    var self = this;
    var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    var url = protocol + '//' + window.location.host + '/ws';

    this.onStatus('connecting');

    this.ws = new WebSocket(url);

    this.ws.onopen = function() {
        console.log('WebSocket connected');
        self.onStatus('connected');
        self.currentDelay = self.reconnectDelay;
    };

    this.ws.onmessage = function(event) {
        try {
            var data = JSON.parse(event.data);
            self.onMessage(data);
        } catch (e) {
            console.error('Failed to parse WS message:', e);
        }
    };

    this.ws.onclose = function() {
        console.log('WebSocket closed, reconnecting in', self.currentDelay, 'ms');
        self.onStatus('disconnected');
        setTimeout(function() {
            self.currentDelay = Math.min(self.currentDelay * 1.5, self.maxReconnectDelay);
            self._connect();
        }, self.currentDelay);
    };

    this.ws.onerror = function() {
        self.ws.close();
    };
};

VMInfoWebSocket.prototype.close = function() {
    if (this.ws) {
        this.ws.close();
    }
};
