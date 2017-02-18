import React, { Component } from 'react';

const ABNORMAL_CLOSURE_ERR = 1006;

class ChatRooms extends Component {
  constructor(props) {
    super(props);
    this.state = {
      chatRooms: [],
      error: false,
      errorMessage: '',
      newRoomName: '',
    }
  }

  componentDidMount() {
    this.fetchRooms();
  }

  fetchRooms = () => {
    fetch('/api/chatroom/list', {
      method: 'GET',
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        response.json().then((responseJson) => {
          this.setState({ chatRooms: responseJson.results });
        });
      }
    });
  };

  showError = (errorMessage) => {
    this.setState({
      error: true,
      errorMessage: errorMessage
    });
  };

  onKeyUp = (event) => {
    this.setState({
      [event.target.className]: event.target.value
    });
  };

  createChatRoom = (event) => {
    event.preventDefault();
    if (!this.state.newRoomName) {
      this.showError('Room name must not be empty');
      return;
    }

    if (!this.props.userName) {
      this.showError('Unable to get username. Please reauthenticate');
      return;
    }

    fetch('/api/chatroom', {
      method: 'POST',
      body: JSON.stringify({
        roomName: this.state.newRoomName,
        createdBy: this.props.userName
      }),
      headers: { 'Content-Type': 'application/json' },
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        this.setState({
          error: false,
          newRoomName: ''
        });
        this.fetchRooms();
      } else {
        response.text().then(this.showError);
      }
    });
  };

  joinChatRoom = (event) => {
    event.preventDefault();

    let roomName = event.target.innerHTML;
    let apiEndpoint = encodeURI('/api/chatroom/' + roomName);

    this.props.createWebSocketConnection(apiEndpoint);
    window.history.pushState({}, '', event.target.href);
  };

  render() {
    let chatRoomList;
    let errorStyling = {color: 'red'};

    if (this.state.chatRooms.length === 0) {
      chatRoomList = <p><i>No chat rooms available. Try creating one above!</i></p>;
    } else {
      let chatRoomLinks = this.state.chatRooms.map((room) => {
        let roomLink = encodeURI('/chatroom/' + room.roomName);
        return (
          <li key={room.id}>
            <a href={roomLink} onClick={this.joinChatRoom}>{room.roomName}</a>
          </li>
        )
      });
      chatRoomList = <ul>{chatRoomLinks}</ul>;
    }

    return (
      <div className="ChatRooms" style={this.props.style}>
        <p>
          <b>Create a new room</b>
        </p>
        <form onSubmit={this.createChatRoom}>
          <input className="newRoomName" type="text" placeholder="Room name" onKeyUp={this.onKeyUp}/>
          <input type="submit"/>
        </form>
        {this.state.error && (
          <div className="errorMessage" style={errorStyling}>{this.state.errorMessage}</div>
        )}
        <p>
          <b>All rooms</b>
        </p>
        {chatRoomList}
      </div>
    );
  }
}

class ChatWindow extends Component {
  constructor(props) {
    super(props);
    this.state = { newMessage: '' };
  }

  setUserInput = (event) => {
    this.setState({ newMessage: event.target.value });
  };

  sendUserMessage = (event) => {
    event.preventDefault();

    let newMessage = this.state.newMessage.trim();
    if (!newMessage) {
      // Don't send empty messages
      return;
    }
    this.props.sendWebSocketChatMessage(newMessage);

    // Clear the input field
    document.querySelector('.userInput').value = '';
  };

  render() {
    let containerStyling = {
      display: 'flex',
      flexDirection: 'column',
    };
    let messagesStyling = {
      flex: 5,
      // This is a hack right now to get scrolling to work
      height: '400px',
      maxHeight: '400px',
      overflowX: 'hidden',
      overflowY: 'scroll',
      wordWrap: 'break-word',
    };
    let textStyling = {
      flex: 1,
      width: '80%'
    };
    let messages = this.props.messages.map((message, index) => {
      return <div key={index}>{message.sentBy + ': ' + message.contents}</div>
    });

    return (
      <div className="ChatWindow" style={this.props.style}>
        <div>Chat here</div>
        <div className="chatContainer" style={containerStyling}>
          <div className="chatMessages" style={messagesStyling}>
            {messages}
          </div>
          <form className="textBox" onSubmit={this.sendUserMessage}>
            <input className="userInput" type="text" style={textStyling} onKeyUp={this.setUserInput}/>
          </form>
        </div>
      </div>
    )
  }
}

class ChatApp extends Component {
  constructor(props) {
    super(props);
    this.state = {
      error: false,
      errorMessage: '',
      messages: [],
      webSocketConn: null,
      userName: '',
    };
  }

  componentDidMount() {
    this.clearError();
    fetch('/user/current', {
      method: 'GET',
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        response.json().then((info) => {
          this.setState({ userName: info.userName });
        });
      } else {
        response.text().then(this.showError);
      }
    });
  }

  clearError = () => {
    this.setState({ error: false });
  };

  showError = (errorMessage) => {
    this.setState({
      error: true,
      errorMessage: errorMessage
    });
  };

  createWebSocketConnection = (relativeUrl) => {
    this.clearError();
    if (this.state.webSocketConn) {
      this.state.webSocketConn.close();
    }

    let webSocket = new WebSocket('wss://' + window.location.host + relativeUrl);
    webSocket.onclose = (event) => {
      if (event.code === ABNORMAL_CLOSURE_ERR) {
        this.showError('Could not connect to chat server');
      }
    };

    webSocket.onmessage = (event) => {
      let response = JSON.parse(event.data);
      if (response.error) {
        this.showError(response.reason);
      }
      this.setState({ messages: this.state.messages.concat(response.body) });
    };

    webSocket.onopen = () => {
      this.setState({ webSocketConn: webSocket });
    };
  };

  sendWebSocketChatMessage = (contents) => {
    let message = {
      contents: contents,
      sentBy: this.state.userName,
      timeSent: new Date(),
    };

    this.state.webSocketConn.send(JSON.stringify(message));
    this.setState({ messages: this.state.messages.concat(message) });
  };

  render() {
    let chatStyling = {
      height: '100%',
      width: '100%'
    };
    let containerStyling = {
      display: 'flex',
      height: '100%',
      width: '100%'
    };
    let chatRoomsStyling = {
      display: 'flex',
      flexDirection: 'column',
      flex: 1
    };
    let chatBoxStyling = { flex: 3 };
    let errorStyling = { color: 'red' };

    return (
      <div className="Chat" style={chatStyling}>
        {this.state.error && (
          <div className="errorMessage" style={errorStyling}>{this.state.errorMessage}</div>
        )}
        <div className="container" style={containerStyling}>
          <ChatRooms
            style={chatRoomsStyling}
            userName={this.state.userName}
            createWebSocketConnection={this.createWebSocketConnection}
          />
          <ChatWindow
            style={chatBoxStyling}
            sendWebSocketChatMessage={this.sendWebSocketChatMessage}
            messages={this.state.messages}
          />
        </div>
      </div>
    );
  }
}

export default ChatApp;
