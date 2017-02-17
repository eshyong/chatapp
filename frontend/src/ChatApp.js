import React, { Component } from 'react';

class ChatRooms extends Component {
  constructor(props) {
    super(props);
    this.state = {
      chatRooms: [],
      error: false,
      errorMessage: '',
      newRoomName: '',
      userName: ''
    }
  }

  componentDidMount() {
    fetch('/api/chatroom/list', {
      method: 'GET',
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        response.json().then((responseJson) => {
          this.setState({
            chatRooms: responseJson.results
          })
        });
      }
    });
  }

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
    console.log(this.state);
  };

  createChatRoom = (event) => {
    event.preventDefault();
    if (!this.state.newRoomName) {
      this.showError('Room name must not be empty');
      return;
    }

    if (!this.state.userName) {
      this.showError('Unable to get username. Please reauthenticate');
      return;
    }

    fetch('/api/chatroom', {
      method: 'POST',
      body: JSON.stringify({
        roomName: this.state.newRoomName,
        createdBy: this.state.userName
      }),
      headers: {
        'Content-Type': 'application/json'
      },
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        this.setState({
          newRoomName: ''
        });
      } else {
        response.text().then(this.showError);
      }
    });
  };

  render() {
    let chatRoomList;
    let errorStyling = {color: 'red'};

    if (this.state.chatRooms.length === 0) {
      chatRoomList = <p><i>No chat rooms available. Try creating one above!</i></p>;
    } else {
      let chatRoomLinks = this.state.chatRooms.map((room) => {
        return (
          <li key={room.id}>{room.roomName}</li>
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
    this.state = {
      newMessage: '',
      messages: [],
    }
  }

  setUserInput = (event) => {
    this.setState({
      newMessage: event.target.value
    });
  };

  sendUserMessage = (event) => {
    event.preventDefault();

    let newMessage = this.state.newMessage.trim();
    if (!newMessage) {
      return;
    }

    let messages = this.state.messages.concat(newMessage);
    this.setState({
      messages: messages,
    });

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
    let messages = this.state.messages.map((message, index) => {
      return <div key={index}>{message}</div>
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
      userName: '',
    };
  }

  componentDidMount() {
    fetch('/user/current', {
      method: 'GET',
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        response.json().then((info) => {
          this.setState({
            userName: info.userName
          });
        });
      } else {
        response.text().then(this.showError);
      }
    });
  }

  showError = (errorMessage) => {
    this.setState({
      error: true,
      errorMessage: errorMessage
    });
  };

  render() {
    let ChatStyling = {
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
    let chatBoxStyling = {
      flex: 3
    };
    let errorStyling = {
      color: 'red'
    };

    return (
      <div className="Chat" style={ChatStyling}>
        {this.state.error && (
          <div className="errorMessage" style={errorStyling}>{this.state.errorMessage}</div>
        )}
        <div className="container" style={containerStyling}>
          <ChatRooms style={chatRoomsStyling}/>
          <ChatWindow style={chatBoxStyling}/>
        </div>
      </div>
    );
  }
}

export default ChatApp;
