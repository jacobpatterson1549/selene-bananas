var game = {
    create: function (event) {
        console.log("TODO: create game");
    },

    join: function (event) {
        console.log("TODO: join game");
    },

    start: function (event) {
        console.log("TODO: send start game request");
    },

    snag: function (event) {
        console.log("TODO: grab request");
    },

    swap: function (event) {
        console.log("TODO: swap tile request"); // how to specify which tile?
    },

    finish: function (event) {
        console.log("TODO: submit game finished request");
    },

    leave: function() {
        if (webSocket) {
            webSocket.close();
        }
    }
};