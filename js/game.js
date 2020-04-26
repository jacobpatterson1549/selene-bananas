var game = {

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
        console.log("TODO: join game");
    },

    delete: function (event) {
        websocket.send({ type: 4 }); // gameDelete
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

    replaceGameTiles: function (unusedTiles, usedTilePositions) {
        this._setTabActive();
        console.log("TODO: set tiles");
    },

    addUnusedTiles: function (unusedTiles) {
        this._setTabActive();
        console.log("TODO: add unused");
    },

    log: function (cls, text) {
        var gameLogItemTemplate = document.getElementById("game-log-item")
        var gameLogItemElement = gameLogItemTemplate.content.cloneNode(true).children[0];
        gameLogItemElement.className = cls;
        var date = new Date();
        var hour = date.getHours();
        var minutes = date.getMinutes()
        var time = hour + ":" + (minutes > 9 ? minutes : "0" + minutes);
        gameLogItemElement.textContent = time + " : " + text;
        var gameLogElement = document.getElementById("game-log");
        var doScroll = gameLogElement.scrollTop >= gameLogElement.scrollHeight - gameLogElement.clientHeight;
        gameLogElement.appendChild(gameLogItemElement);
        if (doScroll) {
            gameLogElement.scrollTop = gameLogElement.scrollHeight - gameLogElement.clientHeight;
        }
    },
};