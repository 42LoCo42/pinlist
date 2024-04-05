async function add(btn) {
	const input = document.querySelector("#newItem");
	const res = await fetch("/add", {
		method: "POST",
		body: input.value,
	});

	if (res.ok) {
		input.value = "";
		location.reload();
	} else {
		alert(`Error: ${await res.text()}`);
	}
}

async function del(btn) {
	const item = btn.closest("tr").querySelector("#item").textContent;
	if (!window.confirm(`really delete ${item}?`)) return;

	const res = await fetch("/del", {
		method: "POST",
		body: item,
	});

	if (res.ok) {
		location.reload();
	} else {
		alert(`Error: ${await res.text()}`);
	}
}
