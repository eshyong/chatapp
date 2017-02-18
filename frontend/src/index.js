import React from 'react';
import ReactDOM from 'react-dom';

import ChatApp from './ChatApp';
import Login from './Login';
import NotFound from './NotFound';

import './index.css';

function main() {
  let path = window.location.pathname;
  let Root = NotFound;

  if (path === '/') {
    Root = <ChatApp/>;
  } else if (path === '/login') {
    Root = <Login/>;
  } else if (path.match(/chatroom\/(\w+)/)) {
    let roomPath = '/api' + path;

    Root = <ChatApp roomPath={roomPath}/>
  }

  ReactDOM.render(Root, document.getElementById('root'));
}

main();
