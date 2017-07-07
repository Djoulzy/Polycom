function StatusGraph(name)
{
	var that = this

	this.newVal1 = 0
	this.newVal2 = 0

	var graphName = "#"+name+"_graph"
	var n = 40,
		random = d3.randomNormal(0, 0),
		data1 = d3.range(n).map(random),
		data2 = d3.range(n).map(random);

	var maxAxisY1 = 50
	var maxAxisY2 = 50

	var svg = d3.select(graphName),
		margin = {top: 20, right: 40, bottom: 20, left: 40},
		width = +svg.attr("width") - margin.left - margin.right,
		height = +svg.attr("height") - margin.top - margin.bottom,
		g = svg.append("g").attr("transform", "translate(" + margin.left + "," + margin.top + ")");
		// g2 = svg.append("g").attr("transform", "translate(" + margin.left + "," + margin.top + ")");

	var xScale = d3.scaleLinear()
		.domain([0, n - 1])
		.range([0, width]);

	var y1Scale = d3.scaleLinear()
		.domain([0, maxAxisY1 - 1])
		.range([height, 0]);

	var y2Scale = d3.scaleLinear()
		.domain([0, maxAxisY2 - 1])
		.range([height, 0]);

	var line1 = d3.line()
		.x(function(d, i) { return xScale(i); })
		.y(function(d, i) { return y1Scale(d); });

	var line2 = d3.line()
		.x(function(d, i) { return xScale(i); })
		.y(function(d, i) { return y2Scale(d); });

	var yAxis1 = d3.axisLeft(y1Scale).ticks(4);
	var yAxis2 = d3.axisRight(y2Scale).ticks(4);

	g.append("defs")
		.append("clipPath")
			.attr("id", "clip")
		.append("rect")
			.attr("width", width)
			.attr("height", height);

	g.append("g")
		.attr("class", "axis y1 "+name)
		.call(yAxis1);

	g.append("g")
		.attr("class", "axis y2 "+name)
		.attr("transform", "translate(" + width + ",0)")
		.call(yAxis2);

	g.append("g")
		.attr("clip-path", "url(#clip)")
			.append("path")
				.datum(data1)
				.attr("class", "draw line1 "+name)
				.transition()
				.duration(5000)
				.ease(d3.easeLinear)
				.on("start", tick);

	g.append("g")
		.attr("clip-path", "url(#clip)")
			.append("path")
				.datum(data2)
				.attr("class", "draw line2 "+name)
				.transition()
				.duration(5000)
				.ease(d3.easeLinear)
				.on("start", tick);

	function tick() {
		// Push a new data point onto the back.
		// console.log(newVal1, newVal2)
		// console.log(data1)
		if (that.newVal1 > maxAxisY1) {
			maxAxisY1 = that.newVal1 + 50
			y1Scale.domain([0, maxAxisY1]);
			g.select(".y1."+name).call(yAxis1);
		}

		if (that.newVal2 > maxAxisY2) {
			maxAxisY2 = that.newVal2 + 50
			g.select(".y2."+name).attr("transform", "translate(" + width + ",0)")
			y2Scale.domain([0, maxAxisY2]);
			g.select(".y2."+name).call(yAxis2);
		}

		data1.push(that.newVal1);
		data2.push(that.newVal2);

		// Redraw the line.
		// d3.select(this)
		d3.select(".line1."+name)
			.attr("d", line1)
			.attr("transform", null);

		d3.select(".line2."+name)
			.attr("d", line2)
			.attr("transform", null);

		// Slide it to the left.
		d3.active(this)
			.attr("transform", "translate(" + xScale(-1) + ",0)")
		.transition()
			.on("start", tick);

		// Pop the old data point off the front.
		data1.shift();
		data2.shift();
	}
}
