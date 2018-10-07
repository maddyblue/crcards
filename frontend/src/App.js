import React, { Component } from 'react';

function shuffle(a) {
	for (let i = a.length - 1; i > 0; i--) {
		const j = Math.floor(Math.random() * (i + 1));
		[a[i], a[j]] = [a[j], a[i]];
	}
	return a;
}

class App extends Component {
	componentDidMount() {
		fetch('/api/get-employees').then(resp => {
			if (resp.status === 200) {
				resp.json().then(data => {
					const teams = {};
					data.forEach(v => {
						teams[v.department] = true;
					});
					this.setState(
						{
							employees: data,
							next: [],
							teams: Object.keys(teams).sort(),
						},
						this.newCard
					);
				});
			}
		});
	}
	newCard() {
		let next = this.state.next;
		if (!next.length) {
			next = shuffle(
				this.state.employees.filter(
					v => !this.state.teamFilter || this.state.teamFilter === v.department
				)
			);
		}
		const emp = next[0];
		next = next.slice(1);
		for (let i = 0; i < next.length && i < 3; i++) {
			// Preload the next few images.
			const tmp = new Image();
			tmp.src = next[i].photoUrl;
		}
		let emps = this.state.employees.filter(
			v =>
				!v.gender || !emp.gender || (v.gender === emp.gender && v.id !== emp.id)
		);
		shuffle(emps);
		emps = emps.slice(0, 3);
		emps.push(emp);
		shuffle(emps);
		this.setState({
			emp: emp,
			emps: emps,
			next: next,
			disabled: {},
		});
	}
	handleClick(v) {
		if (v.id !== this.state.emp.id) {
			const disabled = this.state.disabled;
			disabled[v.id] = true;
			this.setState({
				disabled: disabled,
			});
		} else {
			this.newCard();
		}
	}
	filterTeam = ev => {
		this.setState(
			{
				teamFilter: ev.target.value,
				next: [],
			},
			this.newCard
		);
	};
	render() {
		if (!this.state || !this.state.emp) {
			return 'loading...';
		}
		const emp = this.state.emp;
		const buttons = this.state.emps.map(v => {
			const disabled = this.state.disabled[v.id] === true;
			return (
				<button
					className={'block btn btn-blue' + (disabled ? ' btn-disable' : '')}
					key={v.id}
					disabled={disabled}
					onClick={() => {
						this.handleClick(v);
					}}
				>
					{v.preferredName ? v.preferredName + ' ' + v.lastName : v.displayName}
				</button>
			);
		});
		return (
			<div className="container mx-auto p-2 max-w-md font-sans">
				<div>
					<div className="inline-block align-middle">
						<img
							className="w-150px h-150px"
							src={emp.photoUrl}
							alt={emp.displayName}
						/>
					</div>
					<div className="inline-block align-middle mx-2">
						{emp.department}
						<br />
						{emp.jobTitle}
						<br />
						{emp.location}
					</div>
				</div>
				<div>{buttons}</div>
				<div className="my-8">
					Filter by department:{' '}
					<select onChange={this.filterTeam}>
						<option value="">All Employees</option>
						{this.state.teams.map(v => (
							<option key={v}>{v}</option>
						))}
					</select>
				</div>
				<div className="border-t border-grey py-2 my-8">
					CockroachLabs employee flashcard directory for learning names.
				</div>
			</div>
		);
	}
}

export default App;
