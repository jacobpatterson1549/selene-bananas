var lobby = {

    getGameInfos: function (event) {
        event.preventDefault();
        websocket.connect(event)
            .then(() => {
                websocket.send({ type: 11 }); // gameInfo
            })
            .catch(err => {
                log.error(err);
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
        var timezoneOffsetMinutes = new Date().getTimezoneOffset();
        for (var i = 0; i < gameInfos.length; i++) {
            var gameInfoElement = gameInfoTemplate.content.cloneNode(true);
            var rowElement = gameInfoElement.children[0];
            var createdAt = gameInfos[i].createdAt;
            var createdAtDate = new Date(createdAt); // utc
            createdAtDate.setMinutes(createdAtDate.getMinutes() + timezoneOffsetMinutes);
            var createdAtTime = log.formatDate(createdAtDate);
            rowElement.children[0].innerHTML = createdAtTime;
            rowElement.children[1].innerHTML = gameInfos[i].players;
            this._setStatus(rowElement.children[2], gameInfos[i].status);
            if (gameInfos[i].canJoin) {
                var joinGameButtonTemplate = document.getElementById("join-game-button");
                var joinGameButtonElement = joinGameButtonTemplate.content.cloneNode(true);
                joinGameButtonElement.children[0].value = gameInfos[i].id;
                rowElement.children[2].appendChild(joinGameButtonElement);
            }
            tbodyElement.appendChild(gameInfoElement);
        }

    },

    _setStatus: function (element, status) {
        switch (status) {
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
                log.error("invalid gameStatus: ", status);
                break;
        }
    },

    leave: function () {
        websocket.close();
        game.leave();
    },
};