'use strict';

$(document).ready(function () {
    const enterKey = 13;
    var username = '',
        chatConn;

    $('.username-submit').on('keyup', function(event) {
        var usernameBox = $(event.target),
            chatContainer = $('.chat-container');

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
            console.log('Sorry, this app currently only works on WebSocket-enabled browsers.');
            return;
        }

        chatConn = new window.WebSocket('wss://' + window.location.host + '/chat-room');
        chatConn.onmessage = function(event) {
            var chatMessages = $('.chat-messages'),
                chatWindow = $('.chat-window'),
                message = event.data;

            console.log('Server said: ' + event.data);
            appendChatMessage(message);
            chatWindow.scrollTop(chatMessages.height());
        };
        chatConn.onopen = function() {
            chatConn.send(username);
        }
    }

    $('.chatbox').on('keyup', function(event) {
        var chatMessages = $('.chat-messages'),
            chatWindow = $('.chat-window'),
            chatBox = $(event.target),
            contents = event.target.value.trim(),
            message;

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
        appendChatMessage(message);
        chatWindow.scrollTop(chatMessages.height());
        chatBox.val('');
    });

    function appendChatMessage(message) {
        $('.chat-messages').append('<div class="chat-message">' + message + '</div>')
    }
});
