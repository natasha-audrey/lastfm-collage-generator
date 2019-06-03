import React, { useState } from 'react';
import '../styles/index.scss';

function Home() {
  const [userName, setUserName] = useState('');
  const [size, setSize] = useState('3');
  const [timeframe, setTimeframe] = useState('7day');

  function handleUserNameChange(e) {
    setUserName(e.target.value);
  }

  function handleSizeChange(e) {
    setSize(e.target.value);
  }

  function handleTimeframeChange(e) {
    setTimeframe(e.target.value);
  }

  function handleSubmit(e) {
    if (userName !== '') {
      e.target.action = `/api/v1/${timeframe}/${size}/${userName}`;
    } else {
      e.preventDefault();
    }
  }

  return (
    <div className="container">
      <h1 className="title">Last.fm Collage Generator</h1>
      <p>Generate Last.fm collages from scrobble data!</p>
      <form onSubmit={handleSubmit} id="userForm">
        <label htmlFor="username">
          Last.fm Username
          <br />
          <input onChange={handleUserNameChange} id="username" name="username" value={userName} type="text" />
        </label>
        <select onChange={handleSizeChange} name="size" id="size" value={size}>
          <option value="3">3x3</option>
          <option value="4">4x4</option>
          <option value="5">5x5</option>
          <option value="6">6x6</option>
          <option value="7">7x7</option>
        </select>
        <select onChange={handleTimeframeChange} name="time" id="time" value={timeframe}>
          <option value="7day">Week</option>
          <option value="1month">Month</option>
          <option value="3month">3 Month</option>
          <option value="6month">6 Month</option>
          <option value="12month">Year</option>
          <option value="overall">Overall</option>
        </select>
        <button type="submit">SUBMIT</button>
      </form>
      <div className="bottom">
        <a href="https://github.com/NathanYocum/lastfm-collage-generator">View Source on GitHub!</a>
      </div>
    </div>
  );
}

export default Home;
