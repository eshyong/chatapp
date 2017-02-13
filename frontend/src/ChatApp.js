import React, { Component } from 'react';

class ChatRooms extends Component {
  constructor(props) {
    super(props);
    this.state = {
      chatRooms: [],
      error: false,
      message: '',
      newRoomName: '',
      userName: ''
    }
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
    this.fetchChatRooms();
  }

  fetchChatRooms() {
    fetch('/api/chatroom/list', {
      method: 'GET',
      credentials: 'same-origin'
    })
    .then((response) => {
      if (response.ok) {
        return response.json();
      } else {
        response.text().then(this.showError);
      }
    })
    .then((chatRooms) => {
      this.setState({
        chatRooms: chatRooms.results.map((room) => {
          return {
            name: room.roomName,
            createdBy: room.createdBy
          };
        })
      });
    });
  }

  showError = (message) => {
    this.setState({
      error: true,
      message: message
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
        let newRoom = {
          name: this.state.newRoomName,
          createdBy: this.state.userName
        };
        this.setState({
          newRoomName: '',
          chatRooms: this.state.chatRooms.concat([newRoom])
        });
      } else {
        response.text().then(this.showError);
      }
    });
    this.fetchChatRooms();
  };

  render() {
    let chatRoomList;
    if (this.state.chatRooms.length === 0) {
      chatRoomList = <p><i>No chat rooms available. Try creating one above!</i></p>;
    } else {
      let chatRoomLinks = this.state.chatRooms.map((room) => {
        return (
          <li>{room.name}</li>
        )
      });
      chatRoomList = <ul>{chatRoomLinks}</ul>;
    }
    let errorStyling = {color: 'red'};

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
          <div className="errorMessage" style={errorStyling}>{this.state.message}</div>
        )}
        <p>
          <b>All rooms</b>
        </p>
        {chatRoomList}
      </div>
    );
  }
}

class ChatApp extends Component {
  render() {
    let containerStyling = {
      display: 'flex'
    };
    let chatRoomsStyling = {
      display: 'flex',
      flexDirection: 'column',
      flex: 1
    };
    let chatWindowsStyling = {
      flex: 3
    };

    return (
      <div className="Chat">
        <div className="container" style={containerStyling}>
          <ChatRooms style={chatRoomsStyling}/>
          <div className="chat-window" style={chatWindowsStyling}>
            <p>Chat here</p>
          </div>
        </div>
      </div>
    );
  }
}

export default ChatApp;
