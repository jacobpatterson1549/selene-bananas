var game = {

    unusedTiles: {},
    usedTiles: {},
    usedTileLocs: {},

    create: function (event) {
        websocket.send({ type: 1 }); // gameCreate
    },

    join: function (event) {
        var joinGameButton = event.srcElement;
        var gameIdInput = joinGameButton.previousElementSibling;
        var gameId = parseInt(gameIdInput.value);
        websocket.send({ type: 2, gameID: gameId }); // gameJoin
    },

    leave: function (event) {
        websocket.send({ type: 3 }); // gameLeave
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = false;
        var lobbyTab = document.getElementById("tab-4");
        lobbyTab.checked = true;
    },

    delete: function (event) {
        var result = window.confirm("Are you sure? Deleting the game will kick everyone out.");
        if (!result) {
            return;
        }
        websocket.send({ type: 4 }); // gameDelete
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = false;
        var lobbyTab = document.getElementById("tab-4");
        lobbyTab.checked = true;
    },

    start: function (event) {
        websocket.send({ type: 5, gameState: 1 }); // gameStateChange, gameInProgress
    },

    finish: function (event) {
        websocket.send({ type: 5, gameState: 2 }); // gameStateChange, gameFinished
    },

    snag: function (event) {
        websocket.send({ type: 7 }); // gameSnag
    },

    swap: function (event) {
        console.log("TODO: swap tile request"); // how to specify which tile?
    },

    moveTile: function (event) {
        console.log("TODO: move tile");
    },

    _setTabActive: function () {
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = true;
        var gameTab = document.getElementById("tab-5");
        gameTab.checked = true;
    },

    replaceGameTiles: function (unusedTiles, usedTileLocs) {
        this._setTabActive();
        this.unusedTiles = {}
        this.addUnusedTiles(unusedTiles);
        this.unusedTiles = {}
        this.usedTileLocs = {}
        for (var i = 0; i < usedTileLocs.length; i++) {
            tp = usedTileLocs[i]
            this.usedTiles[t.ID] = tp;
            this.usedTileLocs[tp.x][tp.y] = tp.tile;
        }
        canvas.redraw()
    },

    addUnusedTiles: function (unusedTiles) {
        this._setTabActive();
        var tileStrings = []
        tileStrings.length = unusedTiles.length;
        for (var i = 0; i < unusedTiles.length; i++) {
            var t = unusedTiles[i];
            tileStrings[i] = t.ch;
            this.unusedTiles[t.id] = t;
        }
        tileStrings.sort();
        this.log("info", "adding " + tileStrings + " unused tiles");
        canvas.redraw();
    },

    log: function (cls, text) {
        var gameLogItemTemplate = document.getElementById("game-log-item")
        var gameLogItemElement = gameLogItemTemplate.content.cloneNode(true).children[0];
        gameLogItemElement.className = cls;
        var date = new Date();
        var hour = date.getHours();
        var minutes = date.getMinutes();
        var seconds = date.getSeconds();
        var time = hour + ":" + (minutes > 9 ? minutes : "0" + minutes) + ":" + (seconds > 9 ? seconds : "0" + seconds);
        gameLogItemElement.textContent = time + " : " + text;
        var gameLogElement = document.getElementById("game-log");
        var doScroll = gameLogElement.scrollTop >= gameLogElement.scrollHeight - gameLogElement.clientHeight;
        gameLogElement.appendChild(gameLogItemElement);
        if (doScroll) {
            gameLogElement.scrollTop = gameLogElement.scrollHeight - gameLogElement.clientHeight;
        }
    },
};