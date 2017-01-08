'use strict';

$(document).ready(function () {
    const enterKey = 13;
    var username = '';

    $('.username-submit').on('keyup', function(event) {
        var usernameBox = $(event.target);
        var chatContainer = $('.chat-container');

        if (event.keyCode !== enterKey) {
            return;
        }

        username = usernameBox.val();
        usernameBox.fadeOut();
        chatContainer.fadeIn();
    });

    $('.chatbox').on('keyup', function(event) {
        var chatMessages = $('.chat-messages');
        var chatWindow = $('.chat-window');
        var chatBox = $(event.target);
        var contents = event.target.value.trim('');
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
        chatMessages.append('<div class="chat-message">' + message + '</div>');
        chatWindow.scrollTop(chatMessages.height());
        chatBox.val('');
    });
});
