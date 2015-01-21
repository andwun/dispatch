var React = require('react');
var Reflux = require('reflux');

var selectedTabStore = require('../stores/selectedTab.js');
var messageActions = require('../actions/message.js');
var channelActions = require('../actions/channel.js');

function dispatchCommand(cmd, channel, server) {
	var params = cmd.slice(1).split(' ');

	switch (params[0].toLowerCase()) {
		case 'join':
			if (params[1]) {
				channelActions.join({
					server: server,
					channels: [params[1]]
				});
			}
			break;

		case 'part':
			if (channel) {
				channelActions.part({
					server: server,
					channels: [channel]
				});
			}
			break;
	}
}

var MessageInput = React.createClass({
	mixins: [
		Reflux.connect(selectedTabStore, 'selectedTab')
	],

	getInitialState: function() {
		return {
			selectedTab: selectedTabStore.getState()
		};
	},

	handleKey: function(e) {
		if (e.which === 13 && e.target.value) {
			var tab = this.state.selectedTab;

			if (e.target.value.charAt(0) === '/') {
				dispatchCommand(e.target.value, tab.channel, tab.server);
			} else {
				messageActions.send(e.target.value, tab.channel, tab.server);
			}
			e.target.value = '';
		}
	},

	render: function() {
		return (
			<div className="message-input-wrap">
				<input className="message-input" type="text" onKeyDown={this.handleKey} />
			</div>
		);
	}
});

module.exports = MessageInput;