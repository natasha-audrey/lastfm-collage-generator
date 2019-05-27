const userForm = document.getElementById('userForm');
const usernameInput = document.getElementById('username');
const sizeInput = document.getElementById('size');
const timeInput = document.getElementById('time');

userForm.onsubmit = submit;
usernameInput.onchange = handleUsernameChange;
sizeInput.onchange = handleSizeChange;
timeInput.onchange = handleTimeChange;

let userName = '';
let size = '3';
let time = '7day';

function handleUsernameChange(event) {
  if (event.target.value !== userName) {
    userName = event.target.value;
  }
}

function handleSizeChange(event) {
  if (event.target.value !== size) {
    size = event.target.value;
  }
}

function handleTimeChange(event) {
  if (event.target.value !== time) {
    time = event.target.value;
  }
}

function submit(event) {
  event.target.action = `${event.target.action}${time}/${size}/${userName}`;
  if (userName === '') {
    event.preventDefault()
  }
}
