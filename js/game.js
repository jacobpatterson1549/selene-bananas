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
        console.log("TODO: join game");
    },

    start: function (event) {
        console.log("TODO: send start game request");
    },

    finish: function (event) {
        console.log("TODO: submit game finished request");
    },

    snag: function (event) {
        console.log("TODO: grab request");
    },

    swap: function (event) {
        console.log("TODO: swap tile request"); // how to specify which tile?
    },

    moveTile: function (event) {
        console.log("TODO: move tile");
    },

    setTilePositions: function (unusedTiles, usedTilePositions) {
        console.log("TODO: set tile positions")
    },
};