import React, { Component } from 'react';

class Chat extends Component {
  render() {
    return (
      <div className="Chat">
        <div className="container" style={{display: 'flex'}}>
          <div className="chat-rooms" style={{flex: 1}}>
            <ul>
              <li>Chat room 1</li>
              <li>Chat room 2</li>
              <li>Chat room 3</li>
            </ul>
          </div>
          <div className="chat-window" style={{flex: 3}}>
            <p>Chat here</p>
          </div>
        </div>
      </div>
    );
  }
}

export default Chat;
