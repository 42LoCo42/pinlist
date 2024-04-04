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
	const item = btn.closest("tr").querySelector("#item");
	const res = await fetch("/del", {
		method: "POST",
		body: item.textContent,
	});

	if (res.ok) {
		location.reload();
	} else {
		alert(`Error: ${await res.text()}`);
	}
}
