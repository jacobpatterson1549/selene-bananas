var lobby = {

    getGameInfos: function (event) {
        event.preventDefault();
        websocket.connect(event).then(connected => {
            if (connected) {
                websocket.send({ type: 11 }); // gameInfo
            }
        });
    },

    setGameInfos: function (gameInfos) {
        var gameInfosTable = document.querySelector("table#game-infos")
        var tbodyElement = gameInfosTable.querySelector("tbody");
        tbodyElement.innerHTML = "";
        if (!gameInfos || gameInfos.length == 0) {
            var emptyGameInfoTemplate = document.getElementById("no-game-info-row");
            var emptyGameInfoElement = emptyGameInfoTemplate.content.cloneNode(true);
            tbodyElement.appendChild(emptyGameInfoElement);
            return;
        }
        var gameInfoTemplate = document.getElementById("game-info-row");
        for (i = 0; i < gameInfos.length; i++) {
            var gameInfoElement = gameInfoTemplate.content.cloneNode(true);
            var rowElement = gameInfoElement.children[0];
            rowElement.children[0].innerHTML = gameInfos[i].createdAt;
            rowElement.children[1].innerHTML = gameInfos[i].players;
            this._setState(rowElement.children[2], gameInfos[i].state)
            if (gameInfos[i].canJoin) {
                var joinGameButtonTemplate = document.getElementById("join-game-button");
                var joinGameButtonElement = joinGameButtonTemplate.content.cloneNode(true);
                joinGameButtonElement.children[0].value = gameInfos[i].id;
                rowElement.children[2].appendChild(joinGameButtonElement);
            }
            tbodyElement.appendChild(gameInfoElement);
        }

    },

    _setState: function (element, state) {
        switch (state) {
            case 3: // gameNotStarted
                element.innerHTML = "Not Started"
                break;
            case 1: // gameInProgress
                element.innerHTML = "In Progress"
                break;
            case 2: // gameFinished
                element.innerHTML = "Finished"
                break;
            default:
                log.error("invalid gameState: ", state);
                break;
        }
    },

    leave: function () {
        websocket.close();
        game.leave();
    },
};