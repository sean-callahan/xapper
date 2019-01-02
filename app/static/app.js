
function CreateChannel(id, group, init) {
    var c = document.createElement('div');
    c.className = 'channel';
    c.id = 'channel_' + group + '_' + id;

    var header = document.createElement('p');
    header.className = 'id';
    header.innerText = id;
    c.appendChild(header);

    var name = document.createElement('p');
    name.className = 'name';

    if (init.label === "") {
        name.innerText = PrettyName(group) + ' ' + id;
    } else {
        name.innerText = init.label;
    }
    
    c.appendChild(name);

    var fader = document.createElement('div');
    fader.className = 'fader';

    var level = document.createElement('span');
    level.id = group + '_' + id + '_level';
    level.className = 'level';
    fader.appendChild(level);

    var faderLevel = document.createElement('span');
    faderLevel.id = group + '_' + id + '_faderLevel';
    faderLevel.className = 'faderLevel';
    faderLevel.innerText = init.gain + 'db';
    fader.appendChild(faderLevel);

    var slider = document.createElement('input');
    slider.id = group + '_' + id + '_slider';
    slider.className = 'slider';
    slider.type = 'range';
    slider.min = -65;
    slider.max = 20;
    slider.value = init.gain;
    slider.setAttribute('orient', 'vertical');
    slider.onchange = SliderChanged;
    slider.onclick = function(e) { e.preventDefault(); };
    slider.ondblclick = SliderDoubleClick;
    fader.appendChild(slider);

    var max = document.createElement('span');
    max.className = 'bounds max';
    max.innerText = '20';
    fader.appendChild(max);

    var min = document.createElement('span');
    min.className = 'bounds min';
    min.innerText = '-65';
    fader.appendChild(min);

    c.appendChild(fader);

    var on = document.createElement('button');
    if (init.muted) {
        on.className = 'on';
    } else {
        on.className = 'on lit';
    }
    on.innerText = 'On';
    on.id = group + '_' + id + '_on';
    on.onclick = OnPressed;
    c.appendChild(on);

    var off = document.createElement('button');
    if (init.muted) {
        off.className = 'off lit';
    } else {
        off.className = 'off';
    }
    off.innerText = 'Off';
    off.id = group + '_' + id + '_off';
    off.onclick = OffPressed;
    c.appendChild(off);

    document.getElementById(group).appendChild(c);
}

function PrettyName(group) {
    switch (group) {
        case 'I': return 'Input';
        case 'O': return 'Output';
        default: return group;
    }
}

function SetLevel(type, id, value) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            document.getElementById(type + '_' + id + '_slider').value = this.responseText;
            document.getElementById(type + '_' + id + '_faderLevel').innerText = this.responseText + 'db';
        }
    };
    xhttp.open("GET", 'http://127.0.0.1:1337/0/' + type + '/' + id + '/gain?value=' + value, true);
    xhttp.send();
}

function SliderChanged(e) {
    var parts = e.target.id.split('_');
    var type = parts[0];
    var id = parts[1];

    SetLevel(type, id, e.target.value);
}

function SliderDoubleClick(e) {
    e.preventDefault();

    var parts = e.target.id.split('_');
    var type = parts[0];
    var id = parts[1];

    SetLevel(type, id, 0);
}


function OnPressed(e) {
    var parent = e.target.parentElement;
    var parts = parent.id.split('_');

    var type = parts[1];
    var id = parts[2];

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            document.getElementById(type + '_' + id + '_on').classList.add('lit');
            document.getElementById(type + '_' + id + '_off').classList.remove('lit');
        }
    };
    xhttp.open("GET", 'http://127.0.0.1:1337/0/' + type + '/' + id + '/mute?value=0', true);
    xhttp.send();
}

function OffPressed(e) {
    var parent = e.target.parentElement;
    var parts = parent.id.split('_');

    var type = parts[1];
    var id = parts[2];

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            document.getElementById(type + '_' + id + '_on').classList.remove('lit');
            document.getElementById(type + '_' + id + '_off').classList.add('lit');
        }
    };
    xhttp.open("GET", 'http://127.0.0.1:1337/0/' + type + '/' + id + '/mute?value=1', true);
    xhttp.send();
}

function Heartbeat() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            var state = JSON.parse(this.responseText);
            for (var type in state.channels) {
                var group = state.channels[type];
                for (var i = 0; i < group.length; i++) {
                    var channel = group[i];
                    var value = Math.round(channel.level);
                    var level = document.getElementById(type + '_' + (i+1) + '_level')
                    if (channel.muted) {
                        level.innerText = '';
                        continue;
                    }
                    
                    level.innerText = value + 'db';
                    if (value >= 20) {
                        level.className = 'level red';
                    } else if (value >= 10) {
                        level.className = 'level yellow';
                    } else {
                        level.className = 'level green';
                    }
                }
            }
        }
    };
    xhttp.open("GET", 'http://127.0.0.1:1337/0', true);
    xhttp.send();
}