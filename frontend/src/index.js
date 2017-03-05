import React from 'react';
import ReactDOM from 'react-dom';

import ChatApp from './ChatApp';
import Login from './Login';
import NotFound from './NotFound';

import './index.css';

function main(loggedIn) {
  let path = window.location.pathname;
  let Root = <NotFound/>;

  if (path === '/') {
    Root = loggedIn ? <ChatApp/> : <Login/>;
  } else if (path === '/login') {
    Root = <Login/>;
  } else if (path.match(/chatroom\/([^/]+)/)) {
    let matches = path.match(/chatroom\/([^/]+)/),
        roomName = matches[matches.length - 1];

    Root = <ChatApp roomToJoin={decodeURI(roomName)}/>
  }

  ReactDOM.render(Root, document.getElementById('root'));
}

fetch('/user/current', {
  method: 'GET',
  credentials: 'same-origin',
})
.then((response) => {
  main(response.ok);
});
