const userForm = document.getElementById('userForm');
const usernameInput = document.getElementById('username');

userForm.onsubmit = submit;
usernameInput.onchange = handleChange;

let userName = '';

function handleChange(event) {
  if (event.target.value !== userName) {
    userName = event.target.value;
  }
}

function submit(event) {
  event.target.action = `${event.target.action}${userName}`;
  if (userName === '') {
    event.preventDefault()
  }
}