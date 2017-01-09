'use strict';

$(document).ready(function () {
    const enterKey = 13;
    var username = '';
    var chatConn;

    $('.username-submit').on('keyup', function(event) {
        var usernameBox = $(event.target);
        var chatContainer = $('.chat-container');

        if (event.keyCode !== enterKey) {
            return;
        }

        username = usernameBox.val();
        usernameBox.fadeOut();
        chatContainer.fadeIn();

        createChatConnection();
    });

    function createChatConnection() {
        if (!window.WebSocket) {
            console.log("Your browser doesn't support WebSockets, sorry");
            return;
        }

        chatConn = new window.WebSocket('ws://' + window.location.host + '/chat-room');
        chatConn.onmessage = function(event) {
            var chatMessages = $('.chat-messages');
            var chatWindow = $('.chat-window');
            var message = event.data;

            console.log('Server said: ' + event.data);
            chatMessages.append('<div class="chat-message">' + message + '</div>');
            chatWindow.scrollTop(chatMessages.height());
        };
        chatConn.onopen = function() {
            chatConn.send(username);
        }
    }

    $('.chatbox').on('keyup', function(event) {
        var chatMessages = $('.chat-messages');
        var chatWindow = $('.chat-window');
        var chatBox = $(event.target);
        var contents = event.target.value.trim();
        var message;

        if (event.keyCode !== enterKey) {
            return;
        }

        if (contents === '') {
            // If message box is empty, don't send anything
            return;
        }

        // Display message, scroll to bottom, and clear chatbox
        message = username + ': ' + contents;
        chatConn.send(message);
        chatMessages.append('<div class="chat-message">' + message + '</div>');
        chatWindow.scrollTop(chatMessages.height());
        chatBox.val('');
    });
});
